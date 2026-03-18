// Package blocks implements the `notion blocks` command group.
package blocks

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/neo-notion-cli/internal/cmdutil"
	"github.com/paoloanzn/neo-notion-cli/internal/agents"
	"github.com/paoloanzn/neo-notion-cli/internal/notion"
)

// Cmd returns the `blocks` parent command with its subcommands.
func Cmd() *cobra.Command {
	blocksCmd := &cobra.Command{
		Use:   "blocks",
		Short: "Manage Notion blocks",
		Long:  "Append, retrieve, list children, update, and delete Notion blocks.",
	}

	blocksCmd.AddCommand(appendCmd())
	blocksCmd.AddCommand(getCmd())
	blocksCmd.AddCommand(childrenCmd())
	blocksCmd.AddCommand(updateCmd())
	blocksCmd.AddCommand(deleteCmd())

	return blocksCmd
}

// --- append ---

func appendCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "append",
		Short: "Append child blocks to a parent block",
		Long:  "PATCH /v1/blocks/{block_id}/children — append new children blocks.",
		RunE:  runAppend,
	}
	c.Flags().String("block-id", "", "ID of the parent block (required)")
	_ = c.MarkFlagRequired("block-id")
	c.Flags().String("children", "", "JSON array of child blocks (inline or @file)")
	c.Flags().String("after", "", "Append after this child block ID")
	c.Flags().String("body", "", "Raw JSON request body (overrides other flags)")
	c.Flags().String("body-file", "", "Path to JSON file for request body (overrides other flags)")
	return c
}

func runAppend(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	blockID, _ := c.Flags().GetString("block-id")
	bodyStr, _ := c.Flags().GetString("body")
	bodyFile, _ := c.Flags().GetString("body-file")

	body, err := agents.LoadBody(bodyStr, bodyFile, cfg.Stdin, cfg.InputFile)
	if err != nil {
		return err
	}

	if body == nil {
		// Build body from individual flags.
		childrenStr, _ := c.Flags().GetString("children")
		afterStr, _ := c.Flags().GetString("after")

		if childrenStr == "" {
			return fmt.Errorf("either --body, --body-file, or --children is required")
		}

		childrenJSON, err := agents.LoadJSONOrFile(childrenStr)
		if err != nil {
			return fmt.Errorf("--children: %w", err)
		}

		payload := map[string]interface{}{}

		var childrenVal interface{}
		if err := json.Unmarshal(childrenJSON, &childrenVal); err != nil {
			return fmt.Errorf("--children: invalid JSON: %w", err)
		}
		payload["children"] = childrenVal

		if afterStr != "" {
			payload["after"] = afterStr
		}

		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	data, err := client.AppendBlockChildren(c.Context(), blockID, body)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- get ---

func getCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "get",
		Short: "Retrieve a block by ID",
		Long:  "GET /v1/blocks/{block_id} — retrieve a single block object.",
		RunE:  runGet,
	}
	c.Flags().String("block-id", "", "ID of the block to retrieve (required)")
	_ = c.MarkFlagRequired("block-id")
	return c
}

func runGet(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	blockID, _ := c.Flags().GetString("block-id")

	data, err := client.GetBlock(c.Context(), blockID)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- children ---

func childrenCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "children",
		Short: "List child blocks of a parent block",
		Long:  "GET /v1/blocks/{block_id}/children — list child blocks with pagination.",
		RunE:  runChildren,
	}
	c.Flags().String("block-id", "", "ID of the parent block (required)")
	_ = c.MarkFlagRequired("block-id")
	c.Flags().String("start-cursor", "", "Pagination cursor returned by a previous request")
	c.Flags().Int("page-size", 0, "Number of results per page")
	c.Flags().Bool("recursive", false, "Recursively fetch all nested children (CLI convenience)")
	return c
}

func runChildren(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	blockID, _ := c.Flags().GetString("block-id")
	cursor, _ := c.Flags().GetString("start-cursor")
	pageSize, _ := c.Flags().GetInt("page-size")
	recursive, _ := c.Flags().GetBool("recursive")

	if !recursive {
		data, err := client.GetBlockChildren(c.Context(), blockID, notion.PaginationParams{
			StartCursor: cursor,
			PageSize:    pageSize,
		})
		if err != nil {
			return err
		}
		return cmdutil.OutputResult(cfg, data)
	}

	// Recursive mode: fetch all children, then recurse into blocks that have children.
	allResults, err := fetchChildrenRecursive(c, client, blockID, cursor, pageSize)
	if err != nil {
		return err
	}

	output, err := json.Marshal(map[string]interface{}{
		"object":  "list",
		"results": allResults,
	})
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, output)
}

// fetchChildrenRecursive fetches all children of a block, paginating through
// all pages, then recursively fetches children of any block that has_children.
func fetchChildrenRecursive(c *cobra.Command, client *notion.Client, blockID, startCursor string, pageSize int) ([]interface{}, error) {
	var allResults []interface{}
	cursor := startCursor

	for {
		data, err := client.GetBlockChildren(c.Context(), blockID, notion.PaginationParams{
			StartCursor: cursor,
			PageSize:    pageSize,
		})
		if err != nil {
			return nil, err
		}

		var page struct {
			Results    []json.RawMessage `json:"results"`
			HasMore    bool              `json:"has_more"`
			NextCursor *string           `json:"next_cursor"`
		}
		if err := json.Unmarshal(data, &page); err != nil {
			return nil, fmt.Errorf("parse children response: %w", err)
		}

		for _, raw := range page.Results {
			var block map[string]interface{}
			if err := json.Unmarshal(raw, &block); err != nil {
				return nil, fmt.Errorf("parse block: %w", err)
			}

			// If this block has children, recurse.
			if hasChildren, ok := block["has_children"].(bool); ok && hasChildren {
				id, _ := block["id"].(string)
				if id != "" {
					nested, err := fetchChildrenRecursive(c, client, id, "", pageSize)
					if err != nil {
						return nil, err
					}
					block["children"] = nested
				}
			}

			allResults = append(allResults, block)
		}

		if !page.HasMore || page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}

	return allResults, nil
}

// --- update ---

func updateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "update",
		Short: "Update a block",
		Long:  "PATCH /v1/blocks/{block_id} — update block content or metadata.",
		RunE:  runUpdate,
	}
	c.Flags().String("block-id", "", "ID of the block to update (required)")
	_ = c.MarkFlagRequired("block-id")
	c.Flags().String("payload", "", "JSON object with block type fields (inline or @file)")
	c.Flags().String("in-trash", "", "Set archived/trashed status (true|false)")
	c.Flags().String("body", "", "Raw JSON request body (overrides other flags)")
	c.Flags().String("body-file", "", "Path to JSON file for request body (overrides other flags)")
	return c
}

func runUpdate(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	blockID, _ := c.Flags().GetString("block-id")
	bodyStr, _ := c.Flags().GetString("body")
	bodyFile, _ := c.Flags().GetString("body-file")

	body, err := agents.LoadBody(bodyStr, bodyFile, cfg.Stdin, cfg.InputFile)
	if err != nil {
		return err
	}

	if body == nil {
		// Build body from individual flags.
		payloadStr, _ := c.Flags().GetString("payload")
		inTrashStr, _ := c.Flags().GetString("in-trash")

		if payloadStr == "" && inTrashStr == "" {
			return fmt.Errorf("either --body, --body-file, --payload, or --in-trash is required")
		}

		payload := map[string]interface{}{}

		if payloadStr != "" {
			payloadJSON, err := agents.LoadJSONOrFile(payloadStr)
			if err != nil {
				return fmt.Errorf("--payload: %w", err)
			}
			var payloadMap map[string]interface{}
			if err := json.Unmarshal(payloadJSON, &payloadMap); err != nil {
				return fmt.Errorf("--payload: expected JSON object: %w", err)
			}
			for k, v := range payloadMap {
				payload[k] = v
			}
		}

		if inTrashStr != "" {
			switch inTrashStr {
			case "true":
				payload["in_trash"] = true
			case "false":
				payload["in_trash"] = false
			default:
				return fmt.Errorf("--in-trash must be 'true' or 'false', got %q", inTrashStr)
			}
		}

		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	data, err := client.UpdateBlock(c.Context(), blockID, body)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- delete ---

func deleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete (trash) a block",
		Long:  "DELETE /v1/blocks/{block_id} — move a block to trash.",
		RunE:  runDelete,
	}
	c.Flags().String("block-id", "", "ID of the block to delete (required)")
	_ = c.MarkFlagRequired("block-id")
	return c
}

func runDelete(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	blockID, _ := c.Flags().GetString("block-id")

	data, err := client.DeleteBlock(c.Context(), blockID)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}
