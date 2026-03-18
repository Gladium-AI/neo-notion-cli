// Package cmd defines the Cobra command tree for the notion CLI.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/paoloanzn/neo-notion-cli/internal/cmdutil"
	"github.com/paoloanzn/neo-notion-cli/internal/config"
	"github.com/paoloanzn/neo-notion-cli/internal/notion"

	"github.com/paoloanzn/neo-notion-cli/cmd/auth"
	"github.com/paoloanzn/neo-notion-cli/cmd/blocks"
	"github.com/paoloanzn/neo-notion-cli/cmd/comments"
	"github.com/paoloanzn/neo-notion-cli/cmd/databases"
	"github.com/paoloanzn/neo-notion-cli/cmd/datasources"
	"github.com/paoloanzn/neo-notion-cli/cmd/fileuploads"
	"github.com/paoloanzn/neo-notion-cli/cmd/pages"
	"github.com/paoloanzn/neo-notion-cli/cmd/users"
	"github.com/paoloanzn/neo-notion-cli/cmd/webhooks"
)

var rootCmd = &cobra.Command{
	Use:   "notion",
	Short: "Notion API CLI — agent-first, human-friendly",
	Long: `A CLI that mirrors the Notion public API 1:1.
Designed for AI agents and human operators alike.

Syntax: notion <resource> <verb> [flags]

API version: 2026-03-11`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// searchCmd is a top-level convenience command.
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search by title across pages, databases, and data sources",
	RunE:  runSearch,
}

func init() {
	cobra.OnInitialize(config.InitViper)
	cmdutil.SetRootCmd(rootCmd)

	pf := rootCmd.PersistentFlags()

	// Global flags.
	pf.String("auth-token", "", "Notion API bearer token (env: NOTION_AUTH_TOKEN)")
	pf.String("client-id", "", "OAuth client ID (env: NOTION_CLIENT_ID)")
	pf.String("client-secret", "", "OAuth client secret (env: NOTION_CLIENT_SECRET)")
	pf.String("notion-version", config.DefaultNotionVersion, "Notion API version")
	pf.String("base-url", config.DefaultBaseURL, "Notion API base URL")
	pf.Bool("json", false, "Output as formatted JSON (default)")
	pf.Bool("yaml", false, "Output as YAML")
	pf.Bool("raw", false, "Output raw API response")
	pf.Bool("pretty", false, "Output with pretty formatting")
	pf.Bool("stdin", false, "Read request body from stdin")
	pf.String("input", "", "Read request body from file")
	pf.String("output", "", "Write response to file")
	pf.String("idempotency-key", "", "Idempotency key for mutating requests")
	pf.Duration("timeout", config.DefaultTimeout, "HTTP request timeout")
	pf.Int("retry", config.DefaultRetry, "Max retry count for failed requests")
	pf.StringSlice("header", nil, "Extra headers (k:v)")
	pf.Bool("quiet", false, "Suppress output")
	pf.String("select", "", "jq-like field path to extract from response")
	pf.Bool("paginate", false, "Auto-paginate through all results")
	pf.String("body", "", "Raw JSON body for mutating requests")
	pf.String("body-file", "", "Path to JSON file for request body")

	// Bind all persistent flags to viper.
	_ = viper.BindPFlag("auth-token", pf.Lookup("auth-token"))
	_ = viper.BindPFlag("client-id", pf.Lookup("client-id"))
	_ = viper.BindPFlag("client-secret", pf.Lookup("client-secret"))
	_ = viper.BindPFlag("notion-version", pf.Lookup("notion-version"))
	_ = viper.BindPFlag("base-url", pf.Lookup("base-url"))
	_ = viper.BindPFlag("json", pf.Lookup("json"))
	_ = viper.BindPFlag("yaml", pf.Lookup("yaml"))
	_ = viper.BindPFlag("raw", pf.Lookup("raw"))
	_ = viper.BindPFlag("pretty", pf.Lookup("pretty"))
	_ = viper.BindPFlag("stdin", pf.Lookup("stdin"))
	_ = viper.BindPFlag("input", pf.Lookup("input"))
	_ = viper.BindPFlag("output", pf.Lookup("output"))
	_ = viper.BindPFlag("idempotency-key", pf.Lookup("idempotency-key"))
	_ = viper.BindPFlag("timeout", pf.Lookup("timeout"))
	_ = viper.BindPFlag("retry", pf.Lookup("retry"))
	_ = viper.BindPFlag("quiet", pf.Lookup("quiet"))

	// Search command flags.
	searchCmd.Flags().String("query", "", "Search query text")
	searchCmd.Flags().String("sort-timestamp", "", "Sort by timestamp field (last_edited_time)")
	searchCmd.Flags().String("sort-direction", "", "Sort direction (ascending|descending)")
	searchCmd.Flags().String("filter-property", "", "Filter property name (object)")
	searchCmd.Flags().String("filter-value", "", "Filter value (page|data_source|database)")
	searchCmd.Flags().String("start-cursor", "", "Pagination cursor")
	searchCmd.Flags().Int("page-size", 0, "Results per page")

	// Register subcommands.
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(auth.Cmd())
	rootCmd.AddCommand(users.Cmd())
	rootCmd.AddCommand(pages.Cmd())
	rootCmd.AddCommand(blocks.Cmd())
	rootCmd.AddCommand(databases.Cmd())
	rootCmd.AddCommand(datasources.Cmd())
	rootCmd.AddCommand(comments.Cmd())
	rootCmd.AddCommand(fileuploads.Cmd())
	rootCmd.AddCommand(webhooks.Cmd())
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// NewClientFromConfig creates Notion client from resolved config. Shared by all commands.
func NewClientFromConfig() (*notion.Client, *config.Config, error) {
	return cmdutil.NewClientFromConfig()
}

// OutputResult handles --select and renders the final output.
func OutputResult(cfg *config.Config, data []byte) error {
	return cmdutil.OutputResult(cfg, data)
}

func runSearch(c *cobra.Command, args []string) error {
	client, cfg, err := NewClientFromConfig()
	if err != nil {
		return err
	}

	body := map[string]interface{}{}

	if v, _ := c.Flags().GetString("query"); v != "" {
		body["query"] = v
	}
	if ts, _ := c.Flags().GetString("sort-timestamp"); ts != "" {
		dir, _ := c.Flags().GetString("sort-direction")
		if dir == "" {
			dir = "descending"
		}
		body["sort"] = map[string]string{"timestamp": ts, "direction": dir}
	}
	if fp, _ := c.Flags().GetString("filter-property"); fp != "" {
		fv, _ := c.Flags().GetString("filter-value")
		body["filter"] = map[string]string{"property": fp, "value": fv}
	}
	if v, _ := c.Flags().GetString("start-cursor"); v != "" {
		body["start_cursor"] = v
	}
	if v, _ := c.Flags().GetInt("page-size"); v > 0 {
		body["page_size"] = v
	}

	data, err := client.Search(c.Context(), body)
	if err != nil {
		return err
	}
	return OutputResult(cfg, data)
}
