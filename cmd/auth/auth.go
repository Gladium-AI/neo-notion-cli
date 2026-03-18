// Package auth implements the `notion auth` command group.
package auth

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/paoloanzn/neo-notion-cli/internal/cmdutil"
)

// Cmd returns the `auth` parent command with all subcommands attached.
func Cmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "OAuth authentication endpoints",
	}

	authCmd.AddCommand(tokenCmd())
	return authCmd
}

// tokenCmd returns the `auth token` intermediate command with its subcommands.
func tokenCmd() *cobra.Command {
	token := &cobra.Command{
		Use:   "token",
		Short: "Manage OAuth tokens",
	}

	token.AddCommand(createCmd())
	token.AddCommand(refreshCmd())
	token.AddCommand(introspectCmd())
	token.AddCommand(revokeCmd())
	return token
}

// resolveClientCreds returns the client-id and client-secret, preferring the
// command-local flags and falling back to viper (global flags / env / config).
func resolveClientCreds(c *cobra.Command) (string, string) {
	clientID, _ := c.Flags().GetString("client-id")
	if clientID == "" {
		clientID = viper.GetString("client-id")
	}
	clientSecret, _ := c.Flags().GetString("client-secret")
	if clientSecret == "" {
		clientSecret = viper.GetString("client-secret")
	}
	return clientID, clientSecret
}

// --- notion auth token create ---

func createCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Exchange an authorization code for an access token (POST /v1/oauth/token)",
		RunE:  runCreate,
	}
	c.Flags().String("client-id", "", "OAuth client ID (overrides global)")
	c.Flags().String("client-secret", "", "OAuth client secret (overrides global)")
	c.Flags().String("code", "", "Authorization code from the OAuth callback")
	c.Flags().String("redirect-uri", "", "Redirect URI used in the authorization request")
	c.Flags().String("grant-type", "authorization_code", "Grant type")
	return c
}

func runCreate(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	clientID, clientSecret := resolveClientCreds(c)

	body := map[string]interface{}{}
	if v, _ := c.Flags().GetString("grant-type"); v != "" {
		body["grant_type"] = v
	}
	if v, _ := c.Flags().GetString("code"); v != "" {
		body["code"] = v
	}
	if v, _ := c.Flags().GetString("redirect-uri"); v != "" {
		body["redirect_uri"] = v
	}

	data, err := client.CreateToken(c.Context(), clientID, clientSecret, body)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- notion auth token refresh ---

func refreshCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh an access token (POST /v1/oauth/token)",
		RunE:  runRefresh,
	}
	c.Flags().String("client-id", "", "OAuth client ID (overrides global)")
	c.Flags().String("client-secret", "", "OAuth client secret (overrides global)")
	c.Flags().String("refresh-token", "", "Refresh token")
	c.Flags().String("grant-type", "refresh_token", "Grant type")
	return c
}

func runRefresh(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	clientID, clientSecret := resolveClientCreds(c)

	body := map[string]interface{}{}
	if v, _ := c.Flags().GetString("grant-type"); v != "" {
		body["grant_type"] = v
	}
	if v, _ := c.Flags().GetString("refresh-token"); v != "" {
		body["refresh_token"] = v
	}

	data, err := client.CreateToken(c.Context(), clientID, clientSecret, body)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- notion auth token introspect ---

func introspectCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "introspect",
		Short: "Introspect a token (POST /v1/oauth/introspect)",
		RunE:  runIntrospect,
	}
	c.Flags().String("client-id", "", "OAuth client ID (overrides global)")
	c.Flags().String("client-secret", "", "OAuth client secret (overrides global)")
	c.Flags().String("token", "", "Token to introspect")
	return c
}

func runIntrospect(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	clientID, clientSecret := resolveClientCreds(c)

	body := map[string]interface{}{}
	if v, _ := c.Flags().GetString("token"); v != "" {
		body["token"] = v
	}

	data, err := client.IntrospectToken(c.Context(), clientID, clientSecret, body)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- notion auth token revoke ---

func revokeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke a token (POST /v1/oauth/revoke)",
		RunE:  runRevoke,
	}
	c.Flags().String("client-id", "", "OAuth client ID (overrides global)")
	c.Flags().String("client-secret", "", "OAuth client secret (overrides global)")
	c.Flags().String("token", "", "Token to revoke")
	return c
}

func runRevoke(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	clientID, clientSecret := resolveClientCreds(c)

	body := map[string]interface{}{}
	if v, _ := c.Flags().GetString("token"); v != "" {
		body["token"] = v
	}

	data, err := client.RevokeToken(c.Context(), clientID, clientSecret, body)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}
