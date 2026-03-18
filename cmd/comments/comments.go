// Package comments implements the `notion comments` command group.
package comments

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/paoloanzn/neo-notion-cli/internal/agents"
	"github.com/paoloanzn/neo-notion-cli/internal/cmdutil"
	"github.com/paoloanzn/neo-notion-cli/internal/notion"
)

// Cmd returns the `comments` parent command with its subcommands.
func Cmd() *cobra.Command {
	commentsCmd := &cobra.Command{
		Use:   "comments",
		Short: "Manage Notion comments",
		Long:  "Create, retrieve, and list comments on Notion pages and blocks.",
	}

	commentsCmd.AddCommand(createCmd())
	commentsCmd.AddCommand(getCmd())
	commentsCmd.AddCommand(listCmd())

	return commentsCmd
}

// --- notion comments create ---

func createCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a comment (POST /v1/comments)",
		RunE:  runCreate,
	}
	c.Flags().String("parent-page-id", "", "Page ID to attach the comment to")
	c.Flags().String("discussion-id", "", "Discussion thread ID to reply to")
	c.Flags().String("rich-text", "", "Rich text array as inline JSON or @file path")
	c.Flags().String("body", "", "Raw JSON request body")
	c.Flags().String("body-file", "", "Path to JSON file for request body")
	return c
}

func runCreate(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	bodyFlag, _ := c.Flags().GetString("body")
	bodyFileFlag, _ := c.Flags().GetString("body-file")

	// Try to load a full body from --body, --body-file, --stdin, or --input.
	raw, err := agents.LoadBody(bodyFlag, bodyFileFlag, viper.GetBool("stdin"), viper.GetString("input"))
	if err != nil {
		return err
	}

	// If no full body was provided, build one from individual flags.
	if raw == nil {
		parentPageID, _ := c.Flags().GetString("parent-page-id")
		discussionID, _ := c.Flags().GetString("discussion-id")
		richTextFlag, _ := c.Flags().GetString("rich-text")

		if parentPageID == "" && discussionID == "" {
			return fmt.Errorf("either --parent-page-id or --discussion-id is required")
		}

		payload := map[string]interface{}{}

		if parentPageID != "" {
			payload["parent"] = map[string]string{
				"page_id": parentPageID,
			}
		}
		if discussionID != "" {
			payload["discussion_id"] = discussionID
		}

		if richTextFlag != "" {
			rt, err := agents.LoadJSONOrFile(richTextFlag)
			if err != nil {
				return fmt.Errorf("--rich-text: %w", err)
			}
			payload["rich_text"] = json.RawMessage(rt)
		}

		raw, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
	}

	data, err := client.CreateComment(c.Context(), raw)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- notion comments get ---

func getCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "get",
		Short: "Retrieve a comment by ID (GET /v1/comments/{comment_id})",
		RunE:  runGet,
	}
	c.Flags().String("comment-id", "", "ID of the comment to retrieve")
	_ = c.MarkFlagRequired("comment-id")
	return c
}

func runGet(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	commentID, _ := c.Flags().GetString("comment-id")

	data, err := client.GetComment(c.Context(), commentID)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- notion comments list ---

func listCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List comments for a block (GET /v1/comments?block_id=...)",
		RunE:  runList,
	}
	c.Flags().String("block-id", "", "Block or page ID to list comments for")
	_ = c.MarkFlagRequired("block-id")
	c.Flags().String("start-cursor", "", "Pagination cursor returned by a previous request")
	c.Flags().Int("page-size", 0, "Number of results per page")
	return c
}

func runList(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	blockID, _ := c.Flags().GetString("block-id")
	cursor, _ := c.Flags().GetString("start-cursor")
	pageSize, _ := c.Flags().GetInt("page-size")

	data, err := client.ListComments(c.Context(), blockID, notion.PaginationParams{
		StartCursor: cursor,
		PageSize:    pageSize,
	})
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}
