// Package fileuploads implements the `notion file-uploads` command group.
package fileuploads

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/paoloanzn/neo-notion-cli/internal/cmdutil"
	"github.com/paoloanzn/neo-notion-cli/internal/agents"
	"github.com/paoloanzn/neo-notion-cli/internal/notion"
)

// Cmd returns the `file-uploads` parent command with its subcommands.
func Cmd() *cobra.Command {
	fileUploadsCmd := &cobra.Command{
		Use:   "file-uploads",
		Short: "Manage Notion file uploads",
		Long:  "Create, send, complete, retrieve, and list file uploads.",
	}

	fileUploadsCmd.AddCommand(createCmd())
	fileUploadsCmd.AddCommand(sendCmd())
	fileUploadsCmd.AddCommand(completeCmd())
	fileUploadsCmd.AddCommand(getCmd())
	fileUploadsCmd.AddCommand(listCmd())

	return fileUploadsCmd
}

// --- notion file-uploads create ---

func createCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a file upload (POST /v1/file_uploads)",
		RunE:  runCreate,
	}
	c.Flags().String("filename", "", "Name of the file to upload")
	_ = c.MarkFlagRequired("filename")
	c.Flags().String("content-type", "", "MIME type of the file")
	c.Flags().Int64("content-length", 0, "Size of the file in bytes")
	c.Flags().String("mode", "", "Upload mode (single_part|multi_part)")
	c.Flags().Int("number-of-parts", 0, "Number of parts for multi-part uploads")
	c.Flags().String("body", "", "Raw JSON body (overrides individual flags)")
	c.Flags().String("body-file", "", "Path to JSON file for request body")
	return c
}

func runCreate(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	bodyStr, _ := c.Flags().GetString("body")
	bodyFile, _ := c.Flags().GetString("body-file")
	useStdin := viper.GetBool("stdin")
	inputFile := viper.GetString("input")

	body, err := agents.LoadBody(bodyStr, bodyFile, useStdin, inputFile)
	if err != nil {
		return err
	}

	// If no body was provided via --body/--body-file/stdin, build from flags.
	if body == nil {
		m := map[string]interface{}{}
		if v, _ := c.Flags().GetString("filename"); v != "" {
			m["filename"] = v
		}
		if v, _ := c.Flags().GetString("content-type"); v != "" {
			m["content_type"] = v
		}
		if v, _ := c.Flags().GetInt64("content-length"); v > 0 {
			m["content_length"] = v
		}
		if v, _ := c.Flags().GetString("mode"); v != "" {
			m["mode"] = v
		}
		if v, _ := c.Flags().GetInt("number-of-parts"); v > 0 {
			m["number_of_parts"] = v
		}
		raw, err := json.Marshal(m)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		body = json.RawMessage(raw)
	}

	data, err := client.CreateFileUpload(c.Context(), body)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- notion file-uploads send ---

func sendCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "send",
		Short: "Send file data to an upload (POST /v1/file_uploads/{id}/send)",
		RunE:  runSend,
	}
	c.Flags().String("file-upload-id", "", "ID of the file upload")
	_ = c.MarkFlagRequired("file-upload-id")
	c.Flags().String("file", "", "Path to the file to send")
	_ = c.MarkFlagRequired("file")
	c.Flags().String("content-type", "", "MIME type of the file (auto-detected from extension if omitted)")
	c.Flags().Int("part-number", 0, "Part number for multi-part uploads")
	return c
}

func runSend(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	fileUploadID, _ := c.Flags().GetString("file-upload-id")
	filePath, _ := c.Flags().GetString("file")
	ctOverride, _ := c.Flags().GetString("content-type")
	partNumber, _ := c.Flags().GetInt("part-number")

	// Read the file.
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file %s: %w", filePath, err)
	}

	// Build the multipart form.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add part_number field if specified.
	if partNumber > 0 {
		if err := writer.WriteField("part_number", strconv.Itoa(partNumber)); err != nil {
			return fmt.Errorf("write part_number field: %w", err)
		}
	}

	// Add the file field with the correct MIME type.
	// We cannot use CreateFormFile because it hardcodes
	// Content-Type: application/octet-stream. Notion rejects that
	// when the upload was created with a specific content_type.
	filename := filepath.Base(filePath)
	contentType := ctOverride
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(filePath))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	partHeader.Set("Content-Type", contentType)

	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return fmt.Errorf("create form file part: %w", err)
	}
	if _, err := part.Write(fileData); err != nil {
		return fmt.Errorf("write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	path := fmt.Sprintf("/v1/file_uploads/%s/send", fileUploadID)
	data, err := client.HTTP.DoRaw(c.Context(), "POST", path, &buf, writer.FormDataContentType())
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- notion file-uploads complete ---

func completeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "complete",
		Short: "Complete a file upload (POST /v1/file_uploads/{id}/complete)",
		RunE:  runComplete,
	}
	c.Flags().String("file-upload-id", "", "ID of the file upload to complete")
	_ = c.MarkFlagRequired("file-upload-id")
	return c
}

func runComplete(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	fileUploadID, _ := c.Flags().GetString("file-upload-id")

	data, err := client.CompleteFileUpload(c.Context(), fileUploadID)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- notion file-uploads get ---

func getCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "get",
		Short: "Retrieve a file upload by ID (GET /v1/file_uploads/{id})",
		RunE:  runGet,
	}
	c.Flags().String("file-upload-id", "", "ID of the file upload to retrieve")
	_ = c.MarkFlagRequired("file-upload-id")
	return c
}

func runGet(c *cobra.Command, args []string) error {
	client, cfg, err := cmdutil.NewClientFromConfig()
	if err != nil {
		return err
	}

	fileUploadID, _ := c.Flags().GetString("file-upload-id")

	data, err := client.GetFileUpload(c.Context(), fileUploadID)
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}

// --- notion file-uploads list ---

func listCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List file uploads (GET /v1/file_uploads)",
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

	data, err := client.ListFileUploads(c.Context(), notion.PaginationParams{
		StartCursor: cursor,
		PageSize:    pageSize,
	})
	if err != nil {
		return err
	}
	return cmdutil.OutputResult(cfg, data)
}
