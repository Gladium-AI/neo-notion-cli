package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/neo-notion-cli/internal/cmdutil"
)

func loginCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "login",
		Short: "Save a Notion token (or run the full OAuth browser flow with --oauth)",
		Long: `Saves a Notion integration token to ~/.notion/notion.yaml.

By default, prompts for an internal integration token (ntn_...) or
accepts it via --token. This is the common case for most users.

For public OAuth integrations, pass --oauth along with --client-id and
--client-secret to open the browser consent flow, exchange the code,
and save the resulting token automatically.`,
		RunE: runLogin,
	}
	c.Flags().String("token", "", "Integration token to save (skips interactive prompt)")
	c.Flags().Bool("oauth", false, "Use the full OAuth browser flow (requires --client-id and --client-secret)")
	c.Flags().String("client-id", "", "OAuth client ID (only with --oauth)")
	c.Flags().String("client-secret", "", "OAuth client secret (only with --oauth)")
	c.Flags().Int("port", 9876, "Local port for the OAuth callback server (only with --oauth)")
	c.Flags().String("redirect-uri", "", "Custom redirect URI (only with --oauth)")
	return c
}

func runLogin(c *cobra.Command, args []string) error {
	oauth, _ := c.Flags().GetBool("oauth")
	if oauth {
		return runOAuthLogin(c)
	}
	return runTokenLogin(c)
}

// ---------------------------------------------------------------------------
// Simple token login (default)
// ---------------------------------------------------------------------------

func runTokenLogin(c *cobra.Command, ) error {
	token, _ := c.Flags().GetString("token")

	if token == "" {
		// Interactive prompt.
		fmt.Fprint(os.Stderr, "Paste your Notion internal integration token (ntn_...): ")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read token: %w", err)
		}
		token = strings.TrimSpace(line)
	}

	if token == "" {
		return fmt.Errorf("no token provided")
	}

	// Verify the token works by calling /v1/users/me.
	fmt.Fprint(os.Stderr, "Verifying token... ")
	if err := verifyToken(c, token); err != nil {
		fmt.Fprintln(os.Stderr, "failed")
		return fmt.Errorf("token verification failed: %w", err)
	}
	fmt.Fprintln(os.Stderr, "ok")

	if err := saveToken(token); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Token saved to %s\n", configFilePath())
	return nil
}

// verifyToken calls GET /v1/users/me with the given token to check validity.
func verifyToken(c *cobra.Command, token string) error {
	// Build a one-off client with the provided token rather than config.
	client, _, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}
	// Override the token on the underlying HTTP client.
	client.HTTP.SetAuthToken(token)

	_, err = client.GetMe(c.Context())
	return err
}

// ---------------------------------------------------------------------------
// Full OAuth browser flow (--oauth)
// ---------------------------------------------------------------------------

// tokenResponse is the subset of fields we care about from /v1/oauth/token.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	WorkspaceID  string `json:"workspace_id"`
	BotID        string `json:"bot_id"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func runOAuthLogin(c *cobra.Command) error {
	clientID, clientSecret := resolveClientCreds(c)
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("--client-id and --client-secret are required for --oauth (or set NOTION_CLIENT_ID / NOTION_CLIENT_SECRET)")
	}

	port, _ := c.Flags().GetInt("port")
	redirectURI, _ := c.Flags().GetString("redirect-uri")
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("http://localhost:%d/callback", port)
	}

	// Channel to receive the authorization code (or error) from the callback.
	type callbackResult struct {
		code string
		err  error
	}
	ch := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			desc := r.URL.Query().Get("error_description")
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<html><body><h2>Authorization failed</h2><p>%s: %s</p><p>You can close this tab.</p></body></html>", errParam, desc)
			ch <- callbackResult{err: fmt.Errorf("oauth error: %s — %s", errParam, desc)}
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html><body><h2>Missing authorization code</h2><p>You can close this tab.</p></body></html>")
			ch <- callbackResult{err: fmt.Errorf("callback did not contain an authorization code")}
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><h2>Success!</h2><p>You can close this tab and return to the terminal.</p></body></html>")
		ch <- callbackResult{code: code}
	})

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", port, err)
	}

	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// Build the authorization URL.
	authURL := fmt.Sprintf(
		"https://api.notion.com/v1/oauth/authorize?client_id=%s&response_type=code&owner=user&redirect_uri=%s",
		clientID, redirectURI,
	)

	fmt.Fprintf(os.Stderr, "Opening browser for Notion OAuth login...\n")
	fmt.Fprintf(os.Stderr, "If the browser does not open, visit:\n  %s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser automatically: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Waiting for authorization callback on http://localhost:%d/callback ...\n", port)

	// Wait for the callback.
	result := <-ch
	if result.err != nil {
		return result.err
	}

	fmt.Fprintf(os.Stderr, "Authorization code received. Exchanging for token...\n")

	// Exchange the code for a token.
	client, _, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"grant_type":   "authorization_code",
		"code":         result.code,
		"redirect_uri": redirectURI,
	}

	data, err := client.CreateToken(c.Context(), clientID, clientSecret, body)
	if err != nil {
		return fmt.Errorf("token exchange failed: %w", err)
	}

	var tok tokenResponse
	if err := json.Unmarshal(data, &tok); err != nil {
		return fmt.Errorf("parse token response: %w", err)
	}

	if tok.AccessToken == "" {
		return fmt.Errorf("token exchange returned no access_token: %s", string(data))
	}

	if err := saveToken(tok.AccessToken); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Logged in successfully!\n")
	fmt.Fprintf(os.Stderr, "  Workspace: %s\n", tok.WorkspaceID)
	fmt.Fprintf(os.Stderr, "  Token saved to: %s\n", configFilePath())

	return nil
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// saveToken writes the auth_token to ~/.notion/notion.yaml, preserving
// any other config keys already present.
func saveToken(token string) error {
	cfgPath := configFilePath()

	// Ensure the directory exists.
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	// Read existing config if present.
	existing := map[string]interface{}{}
	if data, err := os.ReadFile(cfgPath); err == nil {
		existing = parseSimpleYAML(data)
	}

	existing["auth_token"] = token

	// Write back as simple YAML.
	f, err := os.OpenFile(cfgPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()

	for k, v := range existing {
		fmt.Fprintf(f, "%s: %q\n", k, v)
	}

	return nil
}

// parseSimpleYAML does a minimal parse of key: "value" or key: value lines.
func parseSimpleYAML(data []byte) map[string]interface{} {
	m := map[string]interface{}{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := line[:idx]
		val := strings.TrimSpace(line[idx+1:])
		// Strip surrounding quotes.
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		if len(val) >= 2 && val[0] == '\'' && val[len(val)-1] == '\'' {
			val = val[1 : len(val)-1]
		}
		if key != "" {
			m[key] = val
		}
	}
	return m
}

func configFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".notion", "notion.yaml")
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
}
