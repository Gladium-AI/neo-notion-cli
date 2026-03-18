// Package render — normalize.go provides a Notion-aware normalization layer
// that transforms verbose API responses into lean, agent-friendly JSON.
//
// Design: the normalizer recognises Notion object types ("page", "list",
// "user", "block", "database", "data_source", "comment") via the "object"
// field and applies type-specific compression.  Unknown shapes pass through
// with only noise-field stripping.
package render

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ---------- public entry point ----------

// Normalize transforms raw Notion JSON into a compact representation.
// Returns the lean JSON bytes or an error.
func Normalize(data []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return data, nil // not JSON — return as-is
	}
	out := normalizeValue(v)
	return json.Marshal(out)
}

// ---------- recursive dispatcher ----------

func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return normalizeObject(val)
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, el := range val {
			out[i] = normalizeValue(el)
		}
		return out
	default:
		return v
	}
}

func normalizeObject(obj map[string]interface{}) interface{} {
	objType, _ := obj["object"].(string)

	switch objType {
	case "list":
		return normalizeList(obj)
	case "page":
		return normalizePage(obj)
	case "user":
		return normalizeUser(obj)
	case "block":
		return normalizeBlock(obj)
	case "database":
		return normalizeDatabase(obj)
	case "data_source":
		return normalizeDataSource(obj)
	case "comment":
		return normalizeComment(obj)
	case "property_item":
		return normalizePropertyItem(obj)
	case "page_markdown":
		return normalizePageMarkdown(obj)
	default:
		return stripNoise(obj)
	}
}

// ---------- list envelope ----------

func normalizeList(obj map[string]interface{}) interface{} {
	out := map[string]interface{}{}

	if results, ok := obj["results"].([]interface{}); ok {
		normalized := make([]interface{}, len(results))
		for i, r := range results {
			normalized[i] = normalizeValue(r)
		}
		out["results"] = normalized
	}

	if hasMore, ok := obj["has_more"]; ok {
		out["has_more"] = hasMore
	}
	if cursor, ok := obj["next_cursor"]; ok && cursor != nil {
		out["next_cursor"] = cursor
	}

	// Preserve property_item list metadata (for paginated property reads).
	if pi, ok := obj["property_item"]; ok {
		out["property_item"] = normalizeValue(pi)
	}

	return out
}

// ---------- page ----------

func normalizePage(obj map[string]interface{}) interface{} {
	out := map[string]interface{}{
		"id":     obj["id"],
		"object": "page",
	}

	// Extract and flatten title from properties.
	if props, ok := obj["properties"].(map[string]interface{}); ok {
		flat := flattenProperties(props)
		out["properties"] = flat

		// Promote the title to a top-level field for convenience.
		for _, v := range props {
			if pm, ok := v.(map[string]interface{}); ok {
				if pm["type"] == "title" {
					out["title"] = flattenProperty(pm)
					break
				}
			}
		}
	}

	// Simplified parent.
	if parent, ok := obj["parent"].(map[string]interface{}); ok {
		out["parent"] = simplifyParent(parent)
	}

	copyIfPresent(out, obj, "url")
	copyIfPresent(out, obj, "created_time")
	copyIfPresent(out, obj, "last_edited_time")
	copyIfPresent(out, obj, "in_trash")

	// Only include if true (noisy when false on every page).
	if locked, ok := obj["is_locked"].(bool); ok && locked {
		out["is_locked"] = true
	}
	if icon := obj["icon"]; icon != nil {
		out["icon"] = icon
	}
	if cover := obj["cover"]; cover != nil {
		out["cover"] = cover
	}

	return out
}

// ---------- user ----------

func normalizeUser(obj map[string]interface{}) interface{} {
	out := map[string]interface{}{
		"id":   obj["id"],
		"name": obj["name"],
		"type": obj["type"],
	}

	// Extract email for person users.
	if person, ok := obj["person"].(map[string]interface{}); ok {
		if email, ok := person["email"].(string); ok && email != "" {
			out["email"] = email
		}
	}

	// For bots, include workspace info.
	if bot, ok := obj["bot"].(map[string]interface{}); ok {
		if wsName, ok := bot["workspace_name"].(string); ok && wsName != "" {
			out["workspace"] = wsName
		}
		if wsID, ok := bot["workspace_id"].(string); ok && wsID != "" {
			out["workspace_id"] = wsID
		}
	}

	return out
}

// ---------- block ----------

func normalizeBlock(obj map[string]interface{}) interface{} {
	out := map[string]interface{}{
		"id":   obj["id"],
		"type": obj["type"],
	}

	blockType, _ := obj["type"].(string)
	if blockType != "" {
		if content, ok := obj[blockType].(map[string]interface{}); ok {
			out["content"] = simplifyBlockContent(blockType, content)
		}
	}

	if hasChildren, ok := obj["has_children"].(bool); ok && hasChildren {
		out["has_children"] = true
	}

	// Preserve recursively-fetched children if present.
	if children, ok := obj["children"]; ok {
		out["children"] = normalizeValue(children)
	}

	copyIfPresent(out, obj, "created_time")
	copyIfPresent(out, obj, "last_edited_time")
	copyIfPresent(out, obj, "in_trash")

	if parent, ok := obj["parent"].(map[string]interface{}); ok {
		out["parent"] = simplifyParent(parent)
	}

	return out
}

func simplifyBlockContent(blockType string, content map[string]interface{}) interface{} {
	// For text-bearing blocks, flatten rich_text to plain string.
	textBlocks := map[string]bool{
		"paragraph": true, "heading_1": true, "heading_2": true, "heading_3": true,
		"bulleted_list_item": true, "numbered_list_item": true, "toggle": true,
		"quote": true, "callout": true, "to_do": true, "code": true,
	}

	if textBlocks[blockType] {
		out := map[string]interface{}{}
		if rt, ok := content["rich_text"]; ok {
			out["text"] = flattenRichText(rt)
		}
		// Preserve special fields per block type.
		if blockType == "to_do" {
			copyIfPresent(out, content, "checked")
		}
		if blockType == "code" {
			copyIfPresent(out, content, "language")
		}
		if blockType == "callout" {
			copyIfPresent(out, content, "icon")
		}
		if len(out) == 1 {
			// Single text field — return just the string.
			if t, ok := out["text"]; ok {
				return t
			}
		}
		return out
	}

	// For other blocks (image, embed, bookmark, etc.) pass through cleaned.
	return stripNoise(content)
}

// ---------- database ----------

func normalizeDatabase(obj map[string]interface{}) interface{} {
	out := map[string]interface{}{
		"id":     obj["id"],
		"object": "database",
	}

	if title := obj["title"]; title != nil {
		out["title"] = flattenRichText(title)
	}
	if desc := obj["description"]; desc != nil {
		d := flattenRichText(desc)
		if d != "" {
			out["description"] = d
		}
	}

	// Simplify the property schema: just name → type (+ options for select/multi_select).
	if props, ok := obj["properties"].(map[string]interface{}); ok {
		schema := map[string]interface{}{}
		for name, val := range props {
			if pm, ok := val.(map[string]interface{}); ok {
				schema[name] = simplifyPropertySchema(pm)
			}
		}
		out["properties"] = schema
	}

	if parent, ok := obj["parent"].(map[string]interface{}); ok {
		out["parent"] = simplifyParent(parent)
	}

	copyIfPresent(out, obj, "url")
	copyIfPresent(out, obj, "created_time")
	copyIfPresent(out, obj, "last_edited_time")
	copyIfPresent(out, obj, "in_trash")

	return out
}

func simplifyPropertySchema(pm map[string]interface{}) interface{} {
	propType, _ := pm["type"].(string)
	out := map[string]interface{}{"type": propType}

	// For select/multi_select/status, include option names.
	switch propType {
	case "select", "status":
		if cfg, ok := pm[propType].(map[string]interface{}); ok {
			if options, ok := cfg["options"].([]interface{}); ok {
				names := make([]string, 0, len(options))
				for _, opt := range options {
					if om, ok := opt.(map[string]interface{}); ok {
						if n, ok := om["name"].(string); ok {
							names = append(names, n)
						}
					}
				}
				if len(names) > 0 {
					out["options"] = names
				}
			}
		}
	case "multi_select":
		if cfg, ok := pm[propType].(map[string]interface{}); ok {
			if options, ok := cfg["options"].([]interface{}); ok {
				names := make([]string, 0, len(options))
				for _, opt := range options {
					if om, ok := opt.(map[string]interface{}); ok {
						if n, ok := om["name"].(string); ok {
							names = append(names, n)
						}
					}
				}
				if len(names) > 0 {
					out["options"] = names
				}
			}
		}
	case "relation":
		if cfg, ok := pm[propType].(map[string]interface{}); ok {
			copyIfPresent(out, cfg, "database_id")
			copyIfPresent(out, cfg, "data_source_id")
		}
	case "formula":
		if cfg, ok := pm[propType].(map[string]interface{}); ok {
			copyIfPresent(out, cfg, "expression")
		}
	case "rollup":
		if cfg, ok := pm[propType].(map[string]interface{}); ok {
			copyIfPresent(out, cfg, "function")
			copyIfPresent(out, cfg, "relation_property_name")
			copyIfPresent(out, cfg, "rollup_property_name")
		}
	}

	// If the only field is type, return just the type string.
	if len(out) == 1 {
		return propType
	}
	return out
}

// ---------- data_source ----------

func normalizeDataSource(obj map[string]interface{}) interface{} {
	out := map[string]interface{}{
		"id":     obj["id"],
		"object": "data_source",
	}

	if title := obj["title"]; title != nil {
		out["title"] = flattenRichText(title)
	}
	if desc := obj["description"]; desc != nil {
		d := flattenRichText(desc)
		if d != "" {
			out["description"] = d
		}
	}

	// Same property schema simplification as database.
	if props, ok := obj["properties"].(map[string]interface{}); ok {
		schema := map[string]interface{}{}
		for name, val := range props {
			if pm, ok := val.(map[string]interface{}); ok {
				schema[name] = simplifyPropertySchema(pm)
			}
		}
		out["properties"] = schema
	}

	if parent, ok := obj["parent"].(map[string]interface{}); ok {
		out["parent"] = simplifyParent(parent)
	}

	copyIfPresent(out, obj, "url")
	copyIfPresent(out, obj, "created_time")
	copyIfPresent(out, obj, "last_edited_time")
	copyIfPresent(out, obj, "in_trash")

	return out
}

// ---------- comment ----------

func normalizeComment(obj map[string]interface{}) interface{} {
	out := map[string]interface{}{
		"id": obj["id"],
	}

	if rt := obj["rich_text"]; rt != nil {
		out["text"] = flattenRichText(rt)
	}
	if parent, ok := obj["parent"].(map[string]interface{}); ok {
		out["parent"] = simplifyParent(parent)
	}
	if user, ok := obj["created_by"].(map[string]interface{}); ok {
		out["created_by"] = normalizeUser(user)
	}
	copyIfPresent(out, obj, "created_time")

	return out
}

// ---------- property_item ----------

func normalizePropertyItem(obj map[string]interface{}) interface{} {
	propType, _ := obj["type"].(string)
	if propType == "" {
		return stripNoise(obj)
	}
	return flattenProperty(obj)
}

// ---------- page_markdown ----------

func normalizePageMarkdown(obj map[string]interface{}) interface{} {
	out := map[string]interface{}{
		"id": obj["id"],
	}
	copyIfPresent(out, obj, "markdown")
	if trunc, ok := obj["truncated"].(bool); ok && trunc {
		out["truncated"] = true
	}
	if ids, ok := obj["unknown_block_ids"].([]interface{}); ok && len(ids) > 0 {
		out["unknown_block_ids"] = ids
	}
	return out
}

// ---------- property value flattening ----------

// flattenProperties converts a full Notion properties map into lean key→value pairs.
func flattenProperties(props map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(props))
	for name, val := range props {
		if pm, ok := val.(map[string]interface{}); ok {
			out[name] = flattenProperty(pm)
		} else {
			out[name] = val
		}
	}
	return out
}

// flattenProperty extracts the scalar value from a typed Notion property.
func flattenProperty(pm map[string]interface{}) interface{} {
	propType, _ := pm["type"].(string)

	switch propType {
	case "title":
		return flattenRichText(pm["title"])

	case "rich_text":
		return flattenRichText(pm["rich_text"])

	case "number":
		return pm["number"] // number or null

	case "select":
		if sel, ok := pm["select"].(map[string]interface{}); ok {
			return sel["name"]
		}
		return nil

	case "multi_select":
		if arr, ok := pm["multi_select"].([]interface{}); ok {
			names := make([]string, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					if n, ok := m["name"].(string); ok {
						names = append(names, n)
					}
				}
			}
			return names
		}
		return nil

	case "status":
		if sel, ok := pm["status"].(map[string]interface{}); ok {
			return sel["name"]
		}
		return nil

	case "date":
		if d, ok := pm["date"].(map[string]interface{}); ok {
			start, _ := d["start"].(string)
			end, _ := d["end"].(string)
			if end != "" {
				return map[string]interface{}{"start": start, "end": end}
			}
			return start // just the start date string
		}
		return nil

	case "checkbox":
		return pm["checkbox"]

	case "url":
		return pm["url"]

	case "email":
		return pm["email"]

	case "phone_number":
		return pm["phone_number"]

	case "formula":
		if f, ok := pm["formula"].(map[string]interface{}); ok {
			fType, _ := f["type"].(string)
			if fType != "" {
				return f[fType]
			}
		}
		return nil

	case "rollup":
		if r, ok := pm["rollup"].(map[string]interface{}); ok {
			rType, _ := r["type"].(string)
			if rType == "array" {
				if arr, ok := r["array"].([]interface{}); ok {
					flat := make([]interface{}, len(arr))
					for i, item := range arr {
						if m, ok := item.(map[string]interface{}); ok {
							flat[i] = flattenProperty(m)
						} else {
							flat[i] = item
						}
					}
					return flat
				}
			}
			if rType != "" {
				return r[rType]
			}
		}
		return nil

	case "people":
		if arr, ok := pm["people"].([]interface{}); ok {
			out := make([]interface{}, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					out = append(out, normalizeUser(m))
				}
			}
			return out
		}
		return nil

	case "files":
		if arr, ok := pm["files"].([]interface{}); ok {
			out := make([]interface{}, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					f := map[string]interface{}{}
					if n, ok := m["name"].(string); ok {
						f["name"] = n
					}
					// External or Notion-hosted file.
					if ext, ok := m["external"].(map[string]interface{}); ok {
						f["url"] = ext["url"]
					} else if file, ok := m["file"].(map[string]interface{}); ok {
						f["url"] = file["url"]
					}
					out = append(out, f)
				}
			}
			return out
		}
		return nil

	case "relation":
		if arr, ok := pm["relation"].([]interface{}); ok {
			ids := make([]string, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					if id, ok := m["id"].(string); ok {
						ids = append(ids, id)
					}
				}
			}
			return ids
		}
		return nil

	case "created_time":
		return pm["created_time"]

	case "last_edited_time":
		return pm["last_edited_time"]

	case "created_by":
		if u, ok := pm["created_by"].(map[string]interface{}); ok {
			return u["id"] // just the ID for brevity
		}
		return nil

	case "last_edited_by":
		if u, ok := pm["last_edited_by"].(map[string]interface{}); ok {
			return u["id"]
		}
		return nil

	case "unique_id":
		if uid, ok := pm["unique_id"].(map[string]interface{}); ok {
			prefix, _ := uid["prefix"].(string)
			number, _ := uid["number"].(float64)
			if prefix != "" {
				return fmt.Sprintf("%s-%d", prefix, int(number))
			}
			return int(number)
		}
		return nil

	case "verification":
		if ver, ok := pm["verification"].(map[string]interface{}); ok {
			return ver["state"]
		}
		return nil

	default:
		// Unknown property type — pass through the typed value.
		if v, ok := pm[propType]; ok {
			return v
		}
		return nil
	}
}

// ---------- rich text ----------

// flattenRichText converts a rich_text array (or any value) to a plain string.
func flattenRichText(v interface{}) string {
	arr, ok := v.([]interface{})
	if !ok {
		return ""
	}
	var sb strings.Builder
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			if pt, ok := m["plain_text"].(string); ok {
				sb.WriteString(pt)
			}
		}
	}
	return sb.String()
}

// ---------- parent ----------

func simplifyParent(parent map[string]interface{}) interface{} {
	pType, _ := parent["type"].(string)
	switch pType {
	case "data_source_id":
		return map[string]interface{}{
			"data_source_id": parent["data_source_id"],
		}
	case "database_id":
		return map[string]interface{}{
			"database_id": parent["database_id"],
		}
	case "page_id":
		return map[string]interface{}{
			"page_id": parent["page_id"],
		}
	case "block_id":
		return map[string]interface{}{
			"block_id": parent["block_id"],
		}
	case "workspace":
		return "workspace"
	default:
		return parent
	}
}

// ---------- generic noise stripping ----------

// noiseFields are top-level keys that add no informational value for agents.
var noiseFields = map[string]bool{
	"request_id": true,
}

func stripNoise(obj map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(obj))
	for k, v := range obj {
		if noiseFields[k] {
			continue
		}
		// Drop null values and empty type-hint objects like "page_or_data_source": {}.
		if v == nil {
			continue
		}
		if m, ok := v.(map[string]interface{}); ok && len(m) == 0 {
			continue
		}
		out[k] = normalizeValue(v)
	}
	return out
}

// ---------- util ----------

func copyIfPresent(dst, src map[string]interface{}, key string) {
	if v, ok := src[key]; ok && v != nil {
		dst[key] = v
	}
}
