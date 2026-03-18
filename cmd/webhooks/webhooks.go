// Package webhooks implements the `notion webhooks` command group.
package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/neo-notion-cli/internal/cmdutil"
)

// Cmd returns the `webhooks` parent command with its subcommands.
func Cmd() *cobra.Command {
	webhooksCmd := &cobra.Command{
		Use:   "webhooks",
		Short: "Webhook utilities (local listener and event metadata)",
		Long:  "Local webhook listener and embedded event-type metadata. No API calls.",
	}

	webhooksCmd.AddCommand(listenCmd())
	webhooksCmd.AddCommand(eventsCmd())

	return webhooksCmd
}

// --- notion webhooks listen ---

func listenCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "listen",
		Short: "Start a local HTTP server to receive Notion webhook events",
		Long: `Starts a local HTTP server that handles the Notion webhook verification
flow and prints incoming webhook event payloads as JSON to stdout.

The server blocks until interrupted (Ctrl-C).`,
		RunE: runListen,
	}
	c.Flags().String("addr", ":8080", "Address to listen on (host:port)")
	c.Flags().String("path", "/notion/webhook", "URL path to handle webhook requests")
	c.Flags().String("verify-token", "", "Verification token from Notion for the one-time challenge")
	return c
}

// verificationRequest represents the JSON body Notion sends during the
// one-time verification flow.
type verificationRequest struct {
	VerificationToken string `json:"verification_token,omitempty"`
	Challenge         string `json:"challenge,omitempty"`
}

// challengeResponse is the JSON response returned for verification requests.
type challengeResponse struct {
	Challenge string `json:"challenge"`
}

func runListen(c *cobra.Command, args []string) error {
	addr, _ := c.Flags().GetString("addr")
	path, _ := c.Flags().GetString("path")
	verifyToken, _ := c.Flags().GetString("verify-token")

	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Try to decode as a verification request.
		var vr verificationRequest
		if err := json.Unmarshal(body, &vr); err == nil && vr.Challenge != "" {
			// This is the one-time verification flow.
			if verifyToken != "" && vr.VerificationToken != verifyToken {
				http.Error(w, "verification token mismatch", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			resp := challengeResponse{Challenge: vr.Challenge}
			_ = json.NewEncoder(w).Encode(resp)
			fmt.Fprintf(os.Stderr, "Verification challenge responded successfully\n")
			return
		}

		// Normal webhook event — print JSON to stdout.
		// Pretty-print if valid JSON, otherwise output raw.
		var pretty json.RawMessage
		if json.Unmarshal(body, &pretty) == nil {
			indented, err := json.MarshalIndent(pretty, "", "  ")
			if err == nil {
				fmt.Fprintln(os.Stdout, string(indented))
			} else {
				fmt.Fprintln(os.Stdout, string(body))
			}
		} else {
			fmt.Fprintln(os.Stdout, string(body))
		}

		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Graceful shutdown on interrupt.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		fmt.Fprintf(os.Stderr, "\nShutting down webhook listener...\n")
		_ = server.Close()
	}()

	fmt.Fprintf(os.Stderr, "Listening for Notion webhooks on %s%s\n", addr, path)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// --- notion webhooks events ---

func eventsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "events",
		Short: "Print supported Notion webhook event types",
		Long:  "Prints a JSON document listing all known Notion webhook event types. No API call is made.",
		RunE:  runEvents,
	}
}

// knownEventTypes is the embedded list of supported Notion webhook event types.
var knownEventTypes = map[string][]string{
	"page": {
		"page.created",
		"page.content_updated",
		"page.properties_updated",
		"page.moved",
		"page.locked",
		"page.unlocked",
		"page.trashed",
		"page.restored",
		"page.permanently_deleted",
		"page.undeleted",
	},
	"database": {
		"database.created",
		"database.updated",
		"database.moved",
		"database.locked",
		"database.unlocked",
		"database.trashed",
		"database.restored",
	},
	"data_source": {
		"data_source.created",
		"data_source.updated",
		"data_source.trashed",
		"data_source.restored",
	},
	"comment": {
		"comment.created",
		"comment.updated",
		"comment.deleted",
	},
}

func runEvents(c *cobra.Command, args []string) error {
	_, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	data, err := json.Marshal(knownEventTypes)
	if err != nil {
		return fmt.Errorf("failed to marshal event types: %w", err)
	}

	return cmdutil.OutputResult(cfg, data)
}
