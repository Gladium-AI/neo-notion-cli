# neo-notion-cli

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Notion API CLI built for AI agents. Ships normalized, context-efficient output by default so agents can read and manipulate Notion workspaces without blowing their context window.

Covers the full Notion API surface (v2026-03-11) — pages, databases, data sources, blocks, comments, file uploads, users, search, OAuth, webhooks — in a single binary.

```
notion <resource> <verb> [flags]
```

## Why this exists

The Notion API returns deeply nested JSON. A single page from a data-source query is ~12 KB of annotations, type wrappers, empty objects, and request metadata. An agent running a handful of queries can burn through its context on boilerplate.

neo-notion-cli fixes this with a **normalization layer** that compresses API responses to their semantic core:

```bash
# Raw API: 12,615 bytes
notion data-sources query --data-source-id $DS --page-size 1 --full

# Normalized (default): 3,582 bytes — same data, 3.5x smaller
notion data-sources query --data-source-id $DS --page-size 1
```

**What gets compressed:**

| Raw API | Normalized |
|---|---|
| `properties.Title.title[0].plain_text` | `"My Page"` |
| `properties.Status.select.name` | `"Done"` |
| `properties.Tags.multi_select[*].name` | `["A", "B"]` |
| `properties.Due.date.start` | `"2025-12-21T..."` |
| `properties.Done.checkbox` | `true` |
| `properties.Assignee.people[*]` | `[{id, name}]` |
| `properties.Linked.relation[*].id` | `["id1", "id2"]` |
| `{object, type, request_id, ...}` list envelope | `{results, has_more, next_cursor}` |
| `annotations: {bold: false, italic: false, ...}` | stripped |
| `created_by: {object: "user", id: "..."}` | stripped (noise on page objects) |

jq expressions via `--select` operate on the normalized output, so agents can do `.title` instead of `.title[0].plain_text`.

## Install

**One-liner (Linux / macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/Gladium-AI/neo-notion-cli/main/install.sh | sh
```

Downloads the latest release for your OS/arch and installs to `~/.local/bin`.

**Go install:**

```bash
go install github.com/paoloanzn/neo-notion-cli@latest
```

**Build from source:**

```bash
git clone https://github.com/Gladium-AI/neo-notion-cli.git
cd neo-notion-cli
make install   # installs to ~/.local/bin
```

## Authentication

**Internal integration (most common):**

1. Create an integration at [notion.so/my-integrations](https://www.notion.so/my-integrations)
2. Copy the token (`ntn_...`)
3. Run:

```bash
notion auth login
# Paste your Notion internal integration token (ntn_...): ████
# Verifying token... ok
# Token saved to ~/.notion/notion.yaml
```

Or set the environment variable:

```bash
export NOTION_AUTH_TOKEN=ntn_...
```

Or pass it per-command:

```bash
notion users me --auth-token ntn_...
```

**Public OAuth integration (for apps others install):**

```bash
notion auth login --oauth --client-id <id> --client-secret <secret>
```

Opens the browser, catches the redirect, exchanges the code, saves the token.

**Token resolution order:** `--auth-token` flag > `NOTION_AUTH_TOKEN` env > `NOTION_TOKEN` env > `~/.notion/notion.yaml` > `./notion.yaml`

## Quick start

```bash
# Who am I?
notion users me

# List all workspace users
notion users list

# Search for pages by title
notion search --query "Q1 Planning"

# Get a page (normalized — flat properties, no noise)
notion pages get --page-id <id>

# Get a page (full API response)
notion pages get --page-id <id> --full

# Query a data source with filters
notion data-sources query --data-source-id <id> \
  --filter '{"property": "Status", "select": {"equals": "Done"}}' \
  --sorts '[{"property": "Created", "direction": "descending"}]' \
  --page-size 10

# Extract specific fields with jq syntax
notion data-sources query --data-source-id <id> --page-size 5 \
  --select '.results[] | {title, status: .properties.Status}'

# Get page content as markdown
notion pages markdown get --page-id <id>

# Create a page from a JSON file
notion pages create --body-file new-page.json

# Pipe JSON from another tool
echo '{"parent":{"page_id":"..."},"properties":{...}}' | notion pages create --stdin
```

## Commands

### Search

```
notion search [--query] [--filter-property --filter-value] [--sort-timestamp --sort-direction] [--page-size] [--start-cursor]
```

`POST /v1/search` — searches by **title** across all pages and data sources the integration can access. Does not search page content or property values.

`--filter-value` accepts `page` or `data_source` (v2026-03-11; `database` is no longer valid).

### Users

```
notion users list [--page-size] [--start-cursor]
notion users get --user-id <id>
notion users me
```

### Pages

```
notion pages create [--body | --body-file | --stdin]
notion pages get --page-id <id>
notion pages update --page-id <id> [--properties | --icon | --cover | --in-trash | --body]
notion pages move --page-id <id> [--parent-page-id | --parent-data-source-id]
notion pages property get --page-id <id> --property-id <id> [--page-size] [--start-cursor]
notion pages markdown get --page-id <id>
notion pages markdown update --page-id <id> --body '...'
notion pages markdown replace --page-id <id> --new-str '...' [--allow-deleting-content]
notion pages markdown insert --page-id <id> --new-str '...' [--after '...']
```

### Blocks

```
notion blocks get --block-id <id>
notion blocks children --block-id <id> [--recursive] [--page-size] [--start-cursor]
notion blocks append --block-id <id> [--body | --body-file | --stdin]
notion blocks update --block-id <id> [--body | --body-file | --stdin]
notion blocks delete --block-id <id>
```

`--recursive` on `children` fetches all nested children, producing a full block tree.

### Databases

```
notion databases create [--body | --body-file | --stdin]
notion databases get --database-id <id>
notion databases update --database-id <id> [--title | --description | --properties | --body]
```

In v2026-03-11, databases hold schema/config. For querying rows and managing property schemas, use **data sources**.

### Data Sources

```
notion data-sources create --database-id <id> --properties '...' [--title]
notion data-sources get --data-source-id <id>
notion data-sources update --data-source-id <id> [--properties | --title | --description | --body]
notion data-sources query --data-source-id <id> [--filter | --sorts | --page-size | --start-cursor]
notion data-sources templates --data-source-id <id>
```

`query` is the primary way to read rows. `--filter` and `--sorts` accept raw JSON matching the Notion filter/sort spec.

### Comments

```
notion comments create [--body | --body-file | --stdin]
notion comments get --comment-id <id>
notion comments list --block-id <id> [--page-size] [--start-cursor]
```

### File Uploads

```
notion file-uploads create --filename <name> [--content-type] [--content-length] [--mode] [--number-of-parts]
notion file-uploads send --file-upload-id <id> --file <path> [--part-number]
notion file-uploads complete --file-upload-id <id>
notion file-uploads get --file-upload-id <id>
notion file-uploads list [--page-size] [--start-cursor]
```

### Auth

```
notion auth login [--token <token>]                    # save an internal integration token
notion auth login --oauth --client-id <id> --client-secret <secret>  # browser OAuth flow
notion auth token create --code <code> --redirect-uri <uri>
notion auth token refresh --refresh-token <token>
notion auth token introspect --token <token>
notion auth token revoke --token <token>
```

### Webhooks

```
notion webhooks listen [--port 8080] [--secret <signing-secret>]
notion webhooks events
```

`listen` starts a local HTTP server that handles Notion's verification challenge and prints incoming events to stdout as JSON.

`events` prints the list of known webhook event types.

## Output control

**Normalization (default):** Responses are compressed to their semantic core. Properties become scalar values, rich text becomes plain strings, list envelopes are stripped of metadata.

```bash
notion users me
# {"id":"...","name":"GLADIUM WORKSPACE INTEGRATION","type":"bot","workspace":"Gladium Agency AI LLC"}
```

**Full API response:** Skip normalization, get the exact API shape (formatted).

```bash
notion users me --full
```

**Raw bytes:** Exact API response, no formatting, no normalization.

```bash
notion users me --raw
```

**YAML:**

```bash
notion users me --yaml
```

**jq-like field selection:** Operates on normalized output.

```bash
notion users me --select '.name'
# "GLADIUM WORKSPACE INTEGRATION"

notion search --page-size 5 --select '.results[] | {id, title}'

notion data-sources query --data-source-id $DS --page-size 3 \
  --select '.results[] | {title, status: .properties.Status, industry: .properties.Industry}'
# [{"title":"...","status":"High","industry":"Legal"}, ...]
```

**Write to file:**

```bash
notion search --output results.json
```

**Suppress output (check exit code only):**

```bash
notion pages update --page-id <id> --body '...' --quiet
```

## Global flags

| Flag | Env var | Description |
|---|---|---|
| `--auth-token` | `NOTION_AUTH_TOKEN` | Bearer token |
| `--client-id` | `NOTION_CLIENT_ID` | OAuth client ID |
| `--client-secret` | `NOTION_CLIENT_SECRET` | OAuth client secret |
| `--notion-version` | | API version (default `2026-03-11`) |
| `--base-url` | | API base URL |
| `--select` | | jq expression to extract fields |
| `--full` | | Skip normalization |
| `--raw` | | Raw API bytes |
| `--json` | | JSON output (default) |
| `--yaml` | | YAML output |
| `--quiet` | | Suppress output |
| `--output` | | Write to file |
| `--body` | | Inline JSON body |
| `--body-file` | | JSON body from file |
| `--stdin` | | Read body from stdin |
| `--paginate` | | Auto-paginate all results |
| `--header` | | Extra HTTP headers (`key:value`) |
| `--idempotency-key` | | Idempotency key for writes |
| `--timeout` | | Request timeout (default `30s`) |
| `--retry` | | Max retries (default `3`) |
| `-v`, `--verbose` | | Debug logging |

## Body input

All mutating commands accept request bodies three ways:

```bash
# Inline JSON
notion pages create --body '{"parent":{"page_id":"..."},"properties":{...}}'

# From a file
notion pages create --body-file create-page.json

# From stdin
cat create-page.json | notion pages create --stdin
```

Priority: `--body` > `--body-file` > `--stdin` > `--input`.

## Configuration

The CLI reads config from (in order):

1. Command-line flags
2. Environment variables (`NOTION_AUTH_TOKEN`, `NOTION_CLIENT_ID`, `NOTION_CLIENT_SECRET`)
3. Config file: `~/.notion/notion.yaml` or `./notion.yaml`

Example config file:

```yaml
auth_token: "ntn_..."
# client_id: "..."
# client_secret: "..."
```

Both underscores (`auth_token`) and hyphens (`auth-token`) work in config files.

## Architecture

```
main.go                      Entry point
cmd/
  root.go                    Root command, global flags, search
  auth/                      OAuth: login, token create/refresh/introspect/revoke
  users/                     Users: list, get, me
  pages/                     Pages: create, get, update, move, property get, markdown *
  blocks/                    Blocks: append, get, children, update, delete
  databases/                 Databases: create, get, update
  datasources/               Data sources: create, get, update, query, templates
  comments/                  Comments: create, get, list
  fileuploads/               File uploads: create, send, complete, get, list
  webhooks/                  Webhooks: listen, events
internal/
  config/                    Layered config (viper): flags > env > file
  httpx/                     Retryable HTTP client with Notion headers
  notion/                    Typed API client (all endpoints), jq select
  render/                    Output formatting + normalization layer
  agents/                    Body loading (--body, --body-file, --stdin)
  cmdutil/                   Shared helpers (breaks import cycle)
```

All HTTP requests go through a single `httpx.Client` that handles:
- Bearer token injection
- `Notion-Version` header
- Retries with exponential backoff (via `go-retryablehttp`)
- Rate limit / `Retry-After` handling
- Structured error parsing into `NotionError{Status, Code, Message}`

## Notion API v2026-03-11

This CLI targets the latest Notion API version which includes:

- **Database / Data Source split**: Databases hold schema configuration. Data sources hold the queryable view of rows. Use `data-sources query` to read rows, not `databases query`.
- **Markdown endpoints**: `pages markdown get/update` for reading and writing page content as markdown instead of block arrays.
- **Search filter**: `--filter-value` accepts `page` or `data_source` (not `database`).

## License

MIT
