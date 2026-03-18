// Package pages implements the `notion pages` command group.
package pages

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/paoloanzn/neo-notion-cli/internal/cmdutil"
	"github.com/paoloanzn/neo-notion-cli/internal/agents"
	"github.com/paoloanzn/neo-notion-cli/internal/notion"
)

// Cmd returns the `pages` parent command with all subcommands registered.
func Cmd() *cobra.Command {
	pagesCmd := &cobra.Command{
		Use:   "pages",
		Short: "Manage Notion pages",
		Long:  "Create, retrieve, update, move pages and work with page properties and markdown content.",
	}

	pagesCmd.AddCommand(createCmd())
	pagesCmd.AddCommand(getCmd())
	pagesCmd.AddCommand(updateCmd())
	pagesCmd.AddCommand(moveCmd())
	pagesCmd.AddCommand(propertyCmd())
	pagesCmd.AddCommand(markdownCmd())

	return pagesCmd
}

// --- pages create ---

func createCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a page (POST /v1/pages)",
		RunE:  runCreate,
	}
	c.Flags().String("parent-page-id", "", "Parent page ID")
	c.Flags().String("parent-data-source-id", "", "Parent data source ID")
	c.Flags().Bool("workspace", false, "Use workspace as parent")
	c.Flags().String("properties", "", "Page properties (JSON string or @file)")
	c.Flags().String("children", "", "Page children blocks (JSON string or @file)")
	c.Flags().String("body", "", "Raw JSON request body")
	c.Flags().String("body-file", "", "Path to JSON file for request body")
	return c
}

func runCreate(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	body, err := resolveBody(c)
	if err != nil {
		return err
	}

	if body == nil {
		m := map[string]interface{}{}

		// Parent.
		parentPageID, _ := c.Flags().GetString("parent-page-id")
		parentDSID, _ := c.Flags().GetString("parent-data-source-id")
		workspace, _ := c.Flags().GetBool("workspace")

		switch {
		case parentPageID != "":
			m["parent"] = map[string]interface{}{"type": "page_id", "page_id": parentPageID}
		case parentDSID != "":
			m["parent"] = map[string]interface{}{"type": "data_source_id", "data_source_id": parentDSID}
		case workspace:
			m["parent"] = map[string]interface{}{"type": "workspace", "workspace": true}
		}

		// Properties.
		if v, _ := c.Flags().GetString("properties"); v != "" {
			raw, err := agents.LoadJSONOrFile(v)
			if err != nil {
				return fmt.Errorf("--properties: %w", err)
			}
			var props interface{}
			if err := json.Unmarshal(raw, &props); err != nil {
				return fmt.Errorf("--properties: %w", err)
			}
			m["properties"] = props
		}

		// Children.
		if v, _ := c.Flags().GetString("children"); v != "" {
			raw, err := agents.LoadJSONOrFile(v)
			if err != nil {
				return fmt.Errorf("--children: %w", err)
			}
			var children interface{}
			if err := json.Unmarshal(raw, &children); err != nil {
				return fmt.Errorf("--children: %w", err)
			}
			m["children"] = children
		}

		body, err = json.Marshal(m)
		if err != nil {
			return err
		}
	}

	data, err := client.CreatePage(c.Context(), body)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- pages get ---

func getCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "get",
		Short: "Retrieve a page (GET /v1/pages/{page_id})",
		RunE:  runGet,
	}
	c.Flags().String("page-id", "", "Page ID (required)")
	_ = c.MarkFlagRequired("page-id")
	return c
}

func runGet(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	pageID, _ := c.Flags().GetString("page-id")

	data, err := client.GetPage(c.Context(), pageID)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- pages update ---

func updateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "update",
		Short: "Update a page (PATCH /v1/pages/{page_id})",
		RunE:  runUpdate,
	}
	c.Flags().String("page-id", "", "Page ID (required)")
	_ = c.MarkFlagRequired("page-id")
	c.Flags().String("properties", "", "Page properties (JSON string or @file)")
	c.Flags().String("icon", "", "Icon object (JSON string or @file)")
	c.Flags().String("cover", "", "Cover object (JSON string or @file)")
	c.Flags().String("is-locked", "", "Lock the page (true|false)")
	c.Flags().String("in-trash", "", "Move to trash (true|false)")
	c.Flags().String("erase-content", "", "Erase page content (true|false)")
	c.Flags().String("body", "", "Raw JSON request body")
	c.Flags().String("body-file", "", "Path to JSON file for request body")
	return c
}

func runUpdate(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	pageID, _ := c.Flags().GetString("page-id")

	body, err := resolveBody(c)
	if err != nil {
		return err
	}

	if body == nil {
		m := map[string]interface{}{}

		if v, _ := c.Flags().GetString("properties"); v != "" {
			raw, err := agents.LoadJSONOrFile(v)
			if err != nil {
				return fmt.Errorf("--properties: %w", err)
			}
			var props interface{}
			if err := json.Unmarshal(raw, &props); err != nil {
				return fmt.Errorf("--properties: %w", err)
			}
			m["properties"] = props
		}

		if v, _ := c.Flags().GetString("icon"); v != "" {
			raw, err := agents.LoadJSONOrFile(v)
			if err != nil {
				return fmt.Errorf("--icon: %w", err)
			}
			var icon interface{}
			if err := json.Unmarshal(raw, &icon); err != nil {
				return fmt.Errorf("--icon: %w", err)
			}
			m["icon"] = icon
		}

		if v, _ := c.Flags().GetString("cover"); v != "" {
			raw, err := agents.LoadJSONOrFile(v)
			if err != nil {
				return fmt.Errorf("--cover: %w", err)
			}
			var cover interface{}
			if err := json.Unmarshal(raw, &cover); err != nil {
				return fmt.Errorf("--cover: %w", err)
			}
			m["cover"] = cover
		}

		if v, _ := c.Flags().GetString("is-locked"); v != "" {
			b, err := parseBoolPtr(v, "--is-locked")
			if err != nil {
				return err
			}
			m["is_locked"] = b
		}

		if v, _ := c.Flags().GetString("in-trash"); v != "" {
			b, err := parseBoolPtr(v, "--in-trash")
			if err != nil {
				return err
			}
			m["in_trash"] = b
		}

		if v, _ := c.Flags().GetString("erase-content"); v != "" {
			b, err := parseBoolPtr(v, "--erase-content")
			if err != nil {
				return err
			}
			m["erase_content"] = b
		}

		body, err = json.Marshal(m)
		if err != nil {
			return err
		}
	}

	data, err := client.UpdatePage(c.Context(), pageID, body)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- pages move ---

func moveCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "move",
		Short: "Move a page (POST /v1/pages/{page_id}/move)",
		RunE:  runMove,
	}
	c.Flags().String("page-id", "", "Page ID (required)")
	_ = c.MarkFlagRequired("page-id")
	c.Flags().String("parent-page-id", "", "New parent page ID")
	c.Flags().String("parent-data-source-id", "", "New parent data source ID")
	c.Flags().String("body", "", "Raw JSON request body")
	c.Flags().String("body-file", "", "Path to JSON file for request body")
	return c
}

func runMove(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	pageID, _ := c.Flags().GetString("page-id")

	body, err := resolveBody(c)
	if err != nil {
		return err
	}

	if body == nil {
		m := map[string]interface{}{}

		parentPageID, _ := c.Flags().GetString("parent-page-id")
		parentDSID, _ := c.Flags().GetString("parent-data-source-id")

		switch {
		case parentPageID != "":
			m["parent"] = map[string]interface{}{"type": "page_id", "page_id": parentPageID}
		case parentDSID != "":
			m["parent"] = map[string]interface{}{"type": "data_source_id", "data_source_id": parentDSID}
		}

		body, err = json.Marshal(m)
		if err != nil {
			return err
		}
	}

	data, err := client.MovePage(c.Context(), pageID, body)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- pages property (group) ---

func propertyCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "property",
		Short: "Work with page properties",
	}
	c.AddCommand(propertyGetCmd())
	return c
}

// --- pages property get ---

func propertyGetCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "get",
		Short: "Retrieve a page property item (GET /v1/pages/{page_id}/properties/{property_id})",
		RunE:  runPropertyGet,
	}
	c.Flags().String("page-id", "", "Page ID (required)")
	_ = c.MarkFlagRequired("page-id")
	c.Flags().String("property-id", "", "Property ID (required)")
	_ = c.MarkFlagRequired("property-id")
	c.Flags().String("start-cursor", "", "Pagination cursor")
	c.Flags().Int("page-size", 0, "Results per page")
	return c
}

func runPropertyGet(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	pageID, _ := c.Flags().GetString("page-id")
	propertyID, _ := c.Flags().GetString("property-id")

	p := notion.PaginationParams{
		StartCursor: mustString(c, "start-cursor"),
		PageSize:    mustInt(c, "page-size"),
	}

	data, err := client.GetPageProperty(c.Context(), pageID, propertyID, p)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- pages markdown (group) ---

func markdownCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "markdown",
		Short: "Work with page markdown content",
	}
	c.AddCommand(markdownGetCmd())
	c.AddCommand(markdownUpdateCmd())
	c.AddCommand(markdownReplaceCmd())
	c.AddCommand(markdownInsertCmd())
	return c
}

// --- pages markdown get ---

func markdownGetCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "get",
		Short: "Get page content as markdown (GET /v1/pages/{page_id}/markdown)",
		RunE:  runMarkdownGet,
	}
	c.Flags().String("page-id", "", "Page ID (required)")
	_ = c.MarkFlagRequired("page-id")
	return c
}

func runMarkdownGet(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	pageID, _ := c.Flags().GetString("page-id")

	data, err := client.GetPageMarkdown(c.Context(), pageID)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- pages markdown update ---

func markdownUpdateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "update",
		Short: "Update page markdown content (PATCH /v1/pages/{page_id}/markdown)",
		RunE:  runMarkdownUpdate,
	}
	c.Flags().String("page-id", "", "Page ID (required)")
	_ = c.MarkFlagRequired("page-id")
	c.Flags().String("body", "", "Raw JSON request body (required unless --body-file)")
	c.Flags().String("body-file", "", "Path to JSON file for request body")
	return c
}

func runMarkdownUpdate(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	pageID, _ := c.Flags().GetString("page-id")

	body, err := resolveBody(c)
	if err != nil {
		return err
	}
	if body == nil {
		return fmt.Errorf("--body or --body-file is required")
	}

	data, err := client.UpdatePageMarkdown(c.Context(), pageID, body)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- pages markdown replace ---

func markdownReplaceCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "replace",
		Short: "Replace page markdown content",
		Long:  "Convenience command that builds an update body with type=replace_content.",
		RunE:  runMarkdownReplace,
	}
	c.Flags().String("page-id", "", "Page ID (required)")
	_ = c.MarkFlagRequired("page-id")
	c.Flags().String("new-str", "", "New markdown content (required)")
	_ = c.MarkFlagRequired("new-str")
	c.Flags().Bool("allow-deleting-content", false, "Allow deleting existing content")
	return c
}

func runMarkdownReplace(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	pageID, _ := c.Flags().GetString("page-id")
	newStr, _ := c.Flags().GetString("new-str")
	allowDelete, _ := c.Flags().GetBool("allow-deleting-content")

	replaceContent := map[string]interface{}{
		"new_str": newStr,
	}
	if allowDelete {
		replaceContent["allow_deleting_content"] = true
	}

	m := map[string]interface{}{
		"type":            "replace_content",
		"replace_content": replaceContent,
	}

	body, err := json.Marshal(m)
	if err != nil {
		return err
	}

	data, err := client.UpdatePageMarkdown(c.Context(), pageID, body)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- pages markdown insert ---

func markdownInsertCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "insert",
		Short: "Insert content into page markdown",
		Long:  "Convenience command that builds an update body with type=insert_content.",
		RunE:  runMarkdownInsert,
	}
	c.Flags().String("page-id", "", "Page ID (required)")
	_ = c.MarkFlagRequired("page-id")
	c.Flags().String("new-str", "", "Markdown content to insert (required)")
	_ = c.MarkFlagRequired("new-str")
	c.Flags().String("after", "", "Selection JSON for insert position")
	return c
}

func runMarkdownInsert(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	pageID, _ := c.Flags().GetString("page-id")
	newStr, _ := c.Flags().GetString("new-str")

	insertContent := map[string]interface{}{
		"new_str": newStr,
	}

	if v, _ := c.Flags().GetString("after"); v != "" {
		raw, err := agents.LoadJSONOrFile(v)
		if err != nil {
			return fmt.Errorf("--after: %w", err)
		}
		var after interface{}
		if err := json.Unmarshal(raw, &after); err != nil {
			return fmt.Errorf("--after: %w", err)
		}
		insertContent["after"] = after
	}

	m := map[string]interface{}{
		"type":           "insert_content",
		"insert_content": insertContent,
	}

	body, err := json.Marshal(m)
	if err != nil {
		return err
	}

	data, err := client.UpdatePageMarkdown(c.Context(), pageID, body)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- helpers ---

// resolveBody loads a raw JSON body from --body or --body-file flags, falling back
// to global --stdin and --input via agents.LoadBody.
func resolveBody(c *cobra.Command) (json.RawMessage, error) {
	bodyStr, _ := c.Flags().GetString("body")
	bodyFile, _ := c.Flags().GetString("body-file")
	useStdin := viper.GetBool("stdin")
	inputFile := viper.GetString("input")
	return agents.LoadBody(bodyStr, bodyFile, useStdin, inputFile)
}

// parseBoolPtr parses a string "true"/"false" into a bool, returning an error
// with the flag name for clarity.
func parseBoolPtr(val, flagName string) (bool, error) {
	switch val {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be 'true' or 'false', got %q", flagName, val)
	}
}

func mustString(c *cobra.Command, name string) string {
	v, _ := c.Flags().GetString(name)
	return v
}

func mustInt(c *cobra.Command, name string) int {
	v, _ := c.Flags().GetInt(name)
	return v
}
