// Package datasources implements the `notion data-sources` command group.
package datasources

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/neo-notion-cli/internal/agents"
	"github.com/paoloanzn/neo-notion-cli/internal/cmdutil"
)

// Cmd returns the `data-sources` parent command with its subcommands.
func Cmd() *cobra.Command {
	dsCmd := &cobra.Command{
		Use:   "data-sources",
		Short: "Manage Notion data sources",
		Long:  "Create, retrieve, update, query, and list templates for Notion data sources.",
	}

	dsCmd.AddCommand(createCmd())
	dsCmd.AddCommand(getCmd())
	dsCmd.AddCommand(updateCmd())
	dsCmd.AddCommand(queryCmd())
	dsCmd.AddCommand(templatesCmd())

	return dsCmd
}

// --- create ---

func createCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a data source",
		RunE:  runCreate,
	}
	c.Flags().String("database-id", "", "ID of the parent database (required)")
	c.Flags().String("properties", "", "Properties JSON or @file (required)")
	c.Flags().String("title", "", "Title JSON or @file")
	c.Flags().String("body", "", "Raw JSON request body")
	c.Flags().String("body-file", "", "Path to JSON file for request body")
	_ = c.MarkFlagRequired("database-id")
	_ = c.MarkFlagRequired("properties")
	return c
}

func runCreate(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	bodyStr, _ := c.Flags().GetString("body")
	bodyFile, _ := c.Flags().GetString("body-file")

	raw, err := agents.LoadBody(bodyStr, bodyFile, cfg.Stdin, cfg.InputFile)
	if err != nil {
		return err
	}

	body := map[string]json.RawMessage{}
	if raw != nil {
		if err := json.Unmarshal(raw, &body); err != nil {
			return err
		}
	}

	dbID, _ := c.Flags().GetString("database-id")
	body["database_id"] = mustMarshal(dbID)

	propsStr, _ := c.Flags().GetString("properties")
	propsJSON, err := agents.LoadJSONOrFile(propsStr)
	if err != nil {
		return err
	}
	if propsJSON != nil {
		body["properties"] = propsJSON
	}

	titleStr, _ := c.Flags().GetString("title")
	titleJSON, err := agents.LoadJSONOrFile(titleStr)
	if err != nil {
		return err
	}
	if titleJSON != nil {
		body["title"] = titleJSON
	}

	merged, err := json.Marshal(body)
	if err != nil {
		return err
	}

	data, err := client.CreateDataSource(c.Context(), json.RawMessage(merged))
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- get ---

func getCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "get",
		Short: "Retrieve a data source by ID",
		RunE:  runGet,
	}
	c.Flags().String("data-source-id", "", "ID of the data source (required)")
	_ = c.MarkFlagRequired("data-source-id")
	return c
}

func runGet(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	dsID, _ := c.Flags().GetString("data-source-id")

	data, err := client.GetDataSource(c.Context(), dsID)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- update ---

func updateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "update",
		Short: "Update a data source",
		RunE:  runUpdate,
	}
	c.Flags().String("data-source-id", "", "ID of the data source (required)")
	c.Flags().String("database-id", "", "ID of the parent database")
	c.Flags().String("title", "", "Title JSON or @file")
	c.Flags().String("description", "", "Description JSON or @file")
	c.Flags().String("properties", "", "Properties JSON or @file")
	c.Flags().String("in-trash", "", "Move to trash (true/false)")
	c.Flags().String("body", "", "Raw JSON request body")
	c.Flags().String("body-file", "", "Path to JSON file for request body")
	_ = c.MarkFlagRequired("data-source-id")
	return c
}

func runUpdate(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	bodyStr, _ := c.Flags().GetString("body")
	bodyFile, _ := c.Flags().GetString("body-file")

	raw, err := agents.LoadBody(bodyStr, bodyFile, cfg.Stdin, cfg.InputFile)
	if err != nil {
		return err
	}

	body := map[string]json.RawMessage{}
	if raw != nil {
		if err := json.Unmarshal(raw, &body); err != nil {
			return err
		}
	}

	if v, _ := c.Flags().GetString("database-id"); v != "" {
		body["database_id"] = mustMarshal(v)
	}

	if v, _ := c.Flags().GetString("title"); v != "" {
		j, err := agents.LoadJSONOrFile(v)
		if err != nil {
			return err
		}
		if j != nil {
			body["title"] = j
		}
	}

	if v, _ := c.Flags().GetString("description"); v != "" {
		j, err := agents.LoadJSONOrFile(v)
		if err != nil {
			return err
		}
		if j != nil {
			body["description"] = j
		}
	}

	if v, _ := c.Flags().GetString("properties"); v != "" {
		j, err := agents.LoadJSONOrFile(v)
		if err != nil {
			return err
		}
		if j != nil {
			body["properties"] = j
		}
	}

	if v, _ := c.Flags().GetString("in-trash"); v != "" {
		if v == "true" {
			body["in_trash"] = json.RawMessage("true")
		} else if v == "false" {
			body["in_trash"] = json.RawMessage("false")
		}
	}

	dsID, _ := c.Flags().GetString("data-source-id")

	merged, err := json.Marshal(body)
	if err != nil {
		return err
	}

	data, err := client.UpdateDataSource(c.Context(), dsID, json.RawMessage(merged))
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- query ---

func queryCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "query",
		Short: "Query a data source",
		RunE:  runQuery,
	}
	c.Flags().String("data-source-id", "", "ID of the data source (required)")
	c.Flags().String("filter", "", "Filter JSON or @file")
	c.Flags().String("sorts", "", "Sorts JSON or @file")
	c.Flags().StringSlice("filter-properties", nil, "Property names to include in response")
	c.Flags().String("start-cursor", "", "Pagination cursor")
	c.Flags().Int("page-size", 0, "Number of results per page")
	c.Flags().String("result-type", "", "Result type filter")
	c.Flags().String("body", "", "Raw JSON request body")
	c.Flags().String("body-file", "", "Path to JSON file for request body")
	_ = c.MarkFlagRequired("data-source-id")
	return c
}

func runQuery(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	bodyStr, _ := c.Flags().GetString("body")
	bodyFile, _ := c.Flags().GetString("body-file")

	raw, err := agents.LoadBody(bodyStr, bodyFile, cfg.Stdin, cfg.InputFile)
	if err != nil {
		return err
	}

	body := map[string]json.RawMessage{}
	if raw != nil {
		if err := json.Unmarshal(raw, &body); err != nil {
			return err
		}
	}

	if v, _ := c.Flags().GetString("filter"); v != "" {
		j, err := agents.LoadJSONOrFile(v)
		if err != nil {
			return err
		}
		if j != nil {
			body["filter"] = j
		}
	}

	if v, _ := c.Flags().GetString("sorts"); v != "" {
		j, err := agents.LoadJSONOrFile(v)
		if err != nil {
			return err
		}
		if j != nil {
			body["sorts"] = j
		}
	}

	if fps, _ := c.Flags().GetStringSlice("filter-properties"); len(fps) > 0 {
		body["filter_properties"] = mustMarshal(fps)
	}

	if v, _ := c.Flags().GetString("start-cursor"); v != "" {
		body["start_cursor"] = mustMarshal(v)
	}

	if v, _ := c.Flags().GetInt("page-size"); v > 0 {
		body["page_size"] = mustMarshal(v)
	}

	if v, _ := c.Flags().GetString("result-type"); v != "" {
		body["result_type"] = mustMarshal(v)
	}

	dsID, _ := c.Flags().GetString("data-source-id")

	merged, err := json.Marshal(body)
	if err != nil {
		return err
	}

	data, err := client.QueryDataSource(c.Context(), dsID, json.RawMessage(merged))
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- templates ---

func templatesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "templates",
		Short: "List templates for a data source",
		RunE:  runTemplates,
	}
	c.Flags().String("data-source-id", "", "ID of the data source (required)")
	_ = c.MarkFlagRequired("data-source-id")
	return c
}

func runTemplates(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	dsID, _ := c.Flags().GetString("data-source-id")

	data, err := client.ListDataSourceTemplates(c.Context(), dsID)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// mustMarshal marshals v to json.RawMessage, panicking on error (safe for simple types).
func mustMarshal(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(b)
}
