// Package users implements the `notion users` command group.
package users

import (
	"github.com/spf13/cobra"

	"github.com/paoloanzn/neo-notion-cli/internal/cmdutil"
	"github.com/paoloanzn/neo-notion-cli/internal/notion"
)

// Cmd returns the `users` parent command with its subcommands.
func Cmd() *cobra.Command {
	usersCmd := &cobra.Command{
		Use:   "users",
		Short: "Manage Notion users",
		Long:  "List, retrieve, and identify Notion workspace users.",
	}

	usersCmd.AddCommand(listCmd())
	usersCmd.AddCommand(getCmd())
	usersCmd.AddCommand(meCmd())

	return usersCmd
}

func listCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List all users in the workspace",
		RunE:  runList,
	}
	c.Flags().String("start-cursor", "", "Pagination cursor returned by a previous request")
	c.Flags().Int("page-size", 0, "Number of results per page")
	return c
}

func runList(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	cursor, _ := c.Flags().GetString("start-cursor")
	pageSize, _ := c.Flags().GetInt("page-size")

	data, err := client.ListUsers(c.Context(), notion.PaginationParams{
		StartCursor: cursor,
		PageSize:    pageSize,
	})
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

func getCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "get",
		Short: "Retrieve a user by ID",
		RunE:  runGet,
	}
	c.Flags().String("user-id", "", "ID of the user to retrieve")
	_ = c.MarkFlagRequired("user-id")
	return c
}

func runGet(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	userID, _ := c.Flags().GetString("user-id")

	data, err := client.GetUser(c.Context(), userID)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

func meCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Get the bot user associated with the current token",
		RunE:  runMe,
	}
}

func runMe(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	data, err := client.GetMe(c.Context())
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}
