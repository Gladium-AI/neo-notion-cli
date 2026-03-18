// Package databases implements the `notion databases` command group.
package databases

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/neo-notion-cli/internal/agents"
	"github.com/paoloanzn/neo-notion-cli/internal/cmdutil"
)

// Cmd returns the `databases` parent command with its subcommands.
func Cmd() *cobra.Command {
	databasesCmd := &cobra.Command{
		Use:   "databases",
		Short: "Manage Notion databases",
		Long:  "Create, retrieve, and update Notion databases.",
	}

	databasesCmd.AddCommand(createCmd())
	databasesCmd.AddCommand(getCmd())
	databasesCmd.AddCommand(updateCmd())

	return databasesCmd
}

// --- create ---

func createCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new database",
		Long:  "Create a database as a child of an existing page (POST /v1/databases).",
		RunE:  runCreate,
	}
	c.Flags().String("parent-page-id", "", "ID of the parent page for the new database")
	_ = c.MarkFlagRequired("parent-page-id")
	c.Flags().String("title", "", "Database title (inline JSON or @file)")
	c.Flags().String("description", "", "Database description (inline JSON or @file)")
	c.Flags().String("icon", "", "Database icon (inline JSON or @file)")
	c.Flags().String("cover", "", "Database cover (inline JSON or @file)")
	c.Flags().Bool("is-inline", false, "Whether the database appears inline in the parent page")
	c.Flags().String("initial-data-source", "", "Initial data source config (inline JSON or @file)")
	c.Flags().String("body", "", "Raw JSON request body (overrides all other flags)")
	c.Flags().String("body-file", "", "Path to JSON file for request body (overrides all other flags)")
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

	if raw == nil {
		body := map[string]interface{}{}

		parentPageID, _ := c.Flags().GetString("parent-page-id")
		body["parent"] = map[string]string{
			"type":    "page_id",
			"page_id": parentPageID,
		}

		if err := setJSONOrFileFlag(c, "title", body); err != nil {
			return err
		}
		if err := setJSONOrFileFlag(c, "description", body); err != nil {
			return err
		}
		if err := setJSONOrFileFlag(c, "icon", body); err != nil {
			return err
		}
		if err := setJSONOrFileFlag(c, "cover", body); err != nil {
			return err
		}
		if err := setJSONOrFileFlag(c, "initial-data-source", body); err != nil {
			return err
		}

		if c.Flags().Changed("is-inline") {
			v, _ := c.Flags().GetBool("is-inline")
			body["is_inline"] = v
		}

		raw, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
	}

	data, err := client.CreateDatabase(c.Context(), raw)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- get ---

func getCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "get",
		Short: "Retrieve a database by ID",
		Long:  "Retrieve a database object by its ID (GET /v1/databases/{database_id}).",
		RunE:  runGet,
	}
	c.Flags().String("database-id", "", "ID of the database to retrieve")
	_ = c.MarkFlagRequired("database-id")
	return c
}

func runGet(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	databaseID, _ := c.Flags().GetString("database-id")

	data, err := client.GetDatabase(c.Context(), databaseID)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- update ---

func updateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "update",
		Short: "Update an existing database",
		Long:  "Update a database's properties, title, description, and more (PATCH /v1/databases/{database_id}).",
		RunE:  runUpdate,
	}
	c.Flags().String("database-id", "", "ID of the database to update")
	_ = c.MarkFlagRequired("database-id")
	c.Flags().String("parent-page-id", "", "New parent page ID (moves the database)")
	c.Flags().Bool("workspace", false, "Set parent to workspace (mutually exclusive with --parent-page-id)")
	c.Flags().String("title", "", "Database title (inline JSON or @file)")
	c.Flags().String("description", "", "Database description (inline JSON or @file)")
	c.Flags().String("icon", "", "Database icon (inline JSON or @file)")
	c.Flags().String("cover", "", "Database cover (inline JSON or @file)")
	c.Flags().String("is-inline", "", "Whether the database is inline (true|false)")
	c.Flags().String("in-trash", "", "Whether the database is in trash (true|false)")
	c.Flags().String("is-locked", "", "Whether the database is locked (true|false)")
	c.Flags().String("body", "", "Raw JSON request body (overrides all other flags)")
	c.Flags().String("body-file", "", "Path to JSON file for request body (overrides all other flags)")
	return c
}

func runUpdate(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	databaseID, _ := c.Flags().GetString("database-id")

	bodyStr, _ := c.Flags().GetString("body")
	bodyFile, _ := c.Flags().GetString("body-file")

	raw, err := agents.LoadBody(bodyStr, bodyFile, cfg.Stdin, cfg.InputFile)
	if err != nil {
		return err
	}

	if raw == nil {
		body := map[string]interface{}{}

		if c.Flags().Changed("parent-page-id") {
			pid, _ := c.Flags().GetString("parent-page-id")
			body["parent"] = map[string]string{
				"type":    "page_id",
				"page_id": pid,
			}
		} else if c.Flags().Changed("workspace") {
			body["parent"] = map[string]interface{}{
				"type":      "workspace",
				"workspace": true,
			}
		}

		if err := setJSONOrFileFlag(c, "title", body); err != nil {
			return err
		}
		if err := setJSONOrFileFlag(c, "description", body); err != nil {
			return err
		}
		if err := setJSONOrFileFlag(c, "icon", body); err != nil {
			return err
		}
		if err := setJSONOrFileFlag(c, "cover", body); err != nil {
			return err
		}

		if err := setBoolPtrFlag(c, "is-inline", "is_inline", body); err != nil {
			return err
		}
		if err := setBoolPtrFlag(c, "in-trash", "in_trash", body); err != nil {
			return err
		}
		if err := setBoolPtrFlag(c, "is-locked", "is_locked", body); err != nil {
			return err
		}

		// Only send a body if at least one field was set.
		if len(body) > 0 {
			raw, err = json.Marshal(body)
			if err != nil {
				return fmt.Errorf("marshal request body: %w", err)
			}
		}
	}

	data, err := client.UpdateDatabase(c.Context(), databaseID, raw)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- helpers ---

// setJSONOrFileFlag reads a flag value that can be inline JSON or @file,
// converts the flag name to a snake_case body key, and sets it on the body map.
func setJSONOrFileFlag(c *cobra.Command, flagName string, body map[string]interface{}) error {
	val, _ := c.Flags().GetString(flagName)
	if val == "" {
		return nil
	}
	parsed, err := agents.LoadJSONOrFile(val)
	if err != nil {
		return fmt.Errorf("--%s: %w", flagName, err)
	}
	if parsed != nil {
		body[flagToKey(flagName)] = json.RawMessage(parsed)
	}
	return nil
}

// setBoolPtrFlag reads a string flag that represents an optional boolean
// ("true"/"false") and sets it on the body map only when explicitly provided.
func setBoolPtrFlag(c *cobra.Command, flagName, bodyKey string, body map[string]interface{}) error {
	if !c.Flags().Changed(flagName) {
		return nil
	}
	val, _ := c.Flags().GetString(flagName)
	switch val {
	case "true":
		body[bodyKey] = true
	case "false":
		body[bodyKey] = false
	default:
		return fmt.Errorf("--%s must be 'true' or 'false', got %q", flagName, val)
	}
	return nil
}

// flagToKey converts a kebab-case flag name to a snake_case JSON body key.
func flagToKey(flag string) string {
	out := make([]byte, len(flag))
	for i := 0; i < len(flag); i++ {
		if flag[i] == '-' {
			out[i] = '_'
		} else {
			out[i] = flag[i]
		}
	}
	return string(out)
}
