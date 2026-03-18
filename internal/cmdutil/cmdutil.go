// Package cmdutil provides shared helpers used by all subcommand packages.
// It exists to break the import cycle between cmd and cmd/<subcommand>.
package cmdutil

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/neo-notion-cli/internal/config"
	"github.com/paoloanzn/neo-notion-cli/internal/httpx"
	"github.com/paoloanzn/neo-notion-cli/internal/notion"
	"github.com/paoloanzn/neo-notion-cli/internal/render"
)

// rootCmd is set by the cmd package at init time via SetRootCmd.
var rootCmd *cobra.Command

// SetRootCmd stores a reference to the root command so helpers can
// read persistent flags without creating an import cycle.
func SetRootCmd(cmd *cobra.Command) {
	rootCmd = cmd
}

// NewClientFromConfig creates a Notion client from the resolved config.
func NewClientFromConfig() (*notion.Client, *config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}

	// Parse extra headers from persistent flags.
	if rootCmd != nil {
		if headers, _ := rootCmd.PersistentFlags().GetStringSlice("header"); len(headers) > 0 {
			cfg.ExtraHeaders = make(map[string]string)
			for _, h := range headers {
				parts := strings.SplitN(h, ":", 2)
				if len(parts) == 2 {
					cfg.ExtraHeaders[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	httpClient := httpx.New(cfg)
	return notion.New(httpClient), cfg, nil
}

// OutputResult normalizes the API response, applies --select, then renders.
//
// Pipeline:
//  1. Normalize (unless --raw or --full)
//  2. Apply --select jq expression (operates on normalized data)
//  3. Render to configured format
//
// When --select is active, render.Output skips re-normalization since the
// data shape has already been transformed by the jq expression.
func OutputResult(cfg *config.Config, data []byte) error {
	sel := ""
	if rootCmd != nil {
		sel, _ = rootCmd.PersistentFlags().GetString("select")
	}

	// Step 1: normalize (unless raw/full mode).
	if !cfg.Full && cfg.OutputFormat != "raw" {
		var err error
		data, err = render.Normalize(data)
		if err != nil {
			return err
		}
	}

	// Step 2: apply jq --select.
	if sel != "" {
		var err error
		data, err = notion.Select(data, sel)
		if err != nil {
			return err
		}
		// After --select, the data is a jq result — skip normalization in render.
		cfg.Full = true
	}

	return render.Output(cfg, data)
}
