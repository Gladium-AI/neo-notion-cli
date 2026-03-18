// Package notion provides typed endpoint clients for the Notion API.
// All methods return raw JSON bytes; callers handle rendering.
package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/paoloanzn/neo-notion-cli/internal/httpx"
)

// Client is the single entry point for all Notion API calls.
type Client struct {
	HTTP *httpx.Client
}

// New creates a Notion client wrapping the given HTTP client.
func New(http *httpx.Client) *Client {
	return &Client{HTTP: http}
}

// --- Pagination helper ---

// PaginationParams holds common pagination flags.
type PaginationParams struct {
	StartCursor string
	PageSize    int
}

func (p PaginationParams) Query() string {
	v := url.Values{}
	if p.StartCursor != "" {
		v.Set("start_cursor", p.StartCursor)
	}
	if p.PageSize > 0 {
		v.Set("page_size", strconv.Itoa(p.PageSize))
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

func (p PaginationParams) Body() map[string]interface{} {
	m := map[string]interface{}{}
	if p.StartCursor != "" {
		m["start_cursor"] = p.StartCursor
	}
	if p.PageSize > 0 {
		m["page_size"] = p.PageSize
	}
	return m
}

// --- Auth / OAuth ---

func (c *Client) CreateToken(ctx context.Context, clientID, clientSecret string, body map[string]interface{}) ([]byte, error) {
	return c.HTTP.DoBasicAuth(ctx, "POST", "/v1/oauth/token", body, clientID, clientSecret)
}

func (c *Client) IntrospectToken(ctx context.Context, clientID, clientSecret string, body map[string]interface{}) ([]byte, error) {
	return c.HTTP.DoBasicAuth(ctx, "POST", "/v1/oauth/introspect", body, clientID, clientSecret)
}

func (c *Client) RevokeToken(ctx context.Context, clientID, clientSecret string, body map[string]interface{}) ([]byte, error) {
	return c.HTTP.DoBasicAuth(ctx, "POST", "/v1/oauth/revoke", body, clientID, clientSecret)
}

// --- Users ---

func (c *Client) ListUsers(ctx context.Context, p PaginationParams) ([]byte, error) {
	return c.HTTP.Do(ctx, "GET", "/v1/users"+p.Query(), nil)
}

func (c *Client) GetUser(ctx context.Context, userID string) ([]byte, error) {
	return c.HTTP.Do(ctx, "GET", "/v1/users/"+userID, nil)
}

func (c *Client) GetMe(ctx context.Context) ([]byte, error) {
	return c.HTTP.Do(ctx, "GET", "/v1/users/me", nil)
}

// --- Search ---

func (c *Client) Search(ctx context.Context, body map[string]interface{}) ([]byte, error) {
	return c.HTTP.Do(ctx, "POST", "/v1/search", body)
}

// --- Pages ---

func (c *Client) CreatePage(ctx context.Context, body json.RawMessage) ([]byte, error) {
	return c.HTTP.Do(ctx, "POST", "/v1/pages", body)
}

func (c *Client) GetPage(ctx context.Context, pageID string) ([]byte, error) {
	return c.HTTP.Do(ctx, "GET", "/v1/pages/"+pageID, nil)
}

func (c *Client) GetPageProperty(ctx context.Context, pageID, propertyID string, p PaginationParams) ([]byte, error) {
	path := fmt.Sprintf("/v1/pages/%s/properties/%s%s", pageID, propertyID, p.Query())
	return c.HTTP.Do(ctx, "GET", path, nil)
}

func (c *Client) UpdatePage(ctx context.Context, pageID string, body json.RawMessage) ([]byte, error) {
	return c.HTTP.Do(ctx, "PATCH", "/v1/pages/"+pageID, body)
}

func (c *Client) MovePage(ctx context.Context, pageID string, body json.RawMessage) ([]byte, error) {
	return c.HTTP.Do(ctx, "POST", "/v1/pages/"+pageID+"/move", body)
}

func (c *Client) GetPageMarkdown(ctx context.Context, pageID string) ([]byte, error) {
	return c.HTTP.Do(ctx, "GET", "/v1/pages/"+pageID+"/markdown", nil)
}

func (c *Client) UpdatePageMarkdown(ctx context.Context, pageID string, body json.RawMessage) ([]byte, error) {
	return c.HTTP.Do(ctx, "PATCH", "/v1/pages/"+pageID+"/markdown", body)
}

// --- Blocks ---

func (c *Client) AppendBlockChildren(ctx context.Context, blockID string, body json.RawMessage) ([]byte, error) {
	return c.HTTP.Do(ctx, "PATCH", "/v1/blocks/"+blockID+"/children", body)
}

func (c *Client) GetBlock(ctx context.Context, blockID string) ([]byte, error) {
	return c.HTTP.Do(ctx, "GET", "/v1/blocks/"+blockID, nil)
}

func (c *Client) GetBlockChildren(ctx context.Context, blockID string, p PaginationParams) ([]byte, error) {
	return c.HTTP.Do(ctx, "GET", "/v1/blocks/"+blockID+"/children"+p.Query(), nil)
}

func (c *Client) UpdateBlock(ctx context.Context, blockID string, body json.RawMessage) ([]byte, error) {
	return c.HTTP.Do(ctx, "PATCH", "/v1/blocks/"+blockID, body)
}

func (c *Client) DeleteBlock(ctx context.Context, blockID string) ([]byte, error) {
	return c.HTTP.Do(ctx, "DELETE", "/v1/blocks/"+blockID, nil)
}

// --- Databases ---

func (c *Client) CreateDatabase(ctx context.Context, body json.RawMessage) ([]byte, error) {
	return c.HTTP.Do(ctx, "POST", "/v1/databases", body)
}

func (c *Client) GetDatabase(ctx context.Context, databaseID string) ([]byte, error) {
	return c.HTTP.Do(ctx, "GET", "/v1/databases/"+databaseID, nil)
}

func (c *Client) UpdateDatabase(ctx context.Context, databaseID string, body json.RawMessage) ([]byte, error) {
	return c.HTTP.Do(ctx, "PATCH", "/v1/databases/"+databaseID, body)
}

// --- Data Sources ---

func (c *Client) CreateDataSource(ctx context.Context, body json.RawMessage) ([]byte, error) {
	return c.HTTP.Do(ctx, "POST", "/v1/data_sources", body)
}

func (c *Client) GetDataSource(ctx context.Context, dataSourceID string) ([]byte, error) {
	return c.HTTP.Do(ctx, "GET", "/v1/data_sources/"+dataSourceID, nil)
}

func (c *Client) UpdateDataSource(ctx context.Context, dataSourceID string, body json.RawMessage) ([]byte, error) {
	return c.HTTP.Do(ctx, "PATCH", "/v1/data_sources/"+dataSourceID, body)
}

func (c *Client) QueryDataSource(ctx context.Context, dataSourceID string, body json.RawMessage) ([]byte, error) {
	return c.HTTP.Do(ctx, "POST", "/v1/data_sources/"+dataSourceID+"/query", body)
}

func (c *Client) ListDataSourceTemplates(ctx context.Context, dataSourceID string) ([]byte, error) {
	return c.HTTP.Do(ctx, "GET", "/v1/data_sources/"+dataSourceID+"/templates", nil)
}

// --- Comments ---

func (c *Client) CreateComment(ctx context.Context, body json.RawMessage) ([]byte, error) {
	return c.HTTP.Do(ctx, "POST", "/v1/comments", body)
}

func (c *Client) GetComment(ctx context.Context, commentID string) ([]byte, error) {
	return c.HTTP.Do(ctx, "GET", "/v1/comments/"+commentID, nil)
}

func (c *Client) ListComments(ctx context.Context, blockID string, p PaginationParams) ([]byte, error) {
	v := url.Values{}
	v.Set("block_id", blockID)
	if p.StartCursor != "" {
		v.Set("start_cursor", p.StartCursor)
	}
	if p.PageSize > 0 {
		v.Set("page_size", strconv.Itoa(p.PageSize))
	}
	return c.HTTP.Do(ctx, "GET", "/v1/comments?"+v.Encode(), nil)
}

// --- File Uploads ---

func (c *Client) CreateFileUpload(ctx context.Context, body json.RawMessage) ([]byte, error) {
	return c.HTTP.Do(ctx, "POST", "/v1/file_uploads", body)
}

func (c *Client) GetFileUpload(ctx context.Context, fileUploadID string) ([]byte, error) {
	return c.HTTP.Do(ctx, "GET", "/v1/file_uploads/"+fileUploadID, nil)
}

func (c *Client) ListFileUploads(ctx context.Context, p PaginationParams) ([]byte, error) {
	return c.HTTP.Do(ctx, "GET", "/v1/file_uploads"+p.Query(), nil)
}

func (c *Client) CompleteFileUpload(ctx context.Context, fileUploadID string) ([]byte, error) {
	return c.HTTP.Do(ctx, "POST", "/v1/file_uploads/"+fileUploadID+"/complete", nil)
}
