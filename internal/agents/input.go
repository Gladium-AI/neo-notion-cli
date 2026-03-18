// Package agents provides utilities for agent-friendly I/O: stdin reading,
// file body loading, and deterministic machine output.
package agents

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// LoadBody resolves a JSON body from --body, --body-file, --stdin, or --input flags.
// Priority: explicit body string > body file > stdin > input file.
func LoadBody(body, bodyFile string, useStdin bool, inputFile string) (json.RawMessage, error) {
	if body != "" {
		if !json.Valid([]byte(body)) {
			return nil, fmt.Errorf("--body value is not valid JSON")
		}
		return json.RawMessage(body), nil
	}

	if bodyFile != "" {
		return readJSONFile(bodyFile)
	}

	if useStdin {
		return readJSONReader(os.Stdin)
	}

	if inputFile != "" {
		return readJSONFile(inputFile)
	}

	return nil, nil
}

// LoadJSONOrFile reads a flag value that can be either inline JSON or a @file path.
func LoadJSONOrFile(value string) (json.RawMessage, error) {
	if value == "" {
		return nil, nil
	}
	if strings.HasPrefix(value, "@") {
		return readJSONFile(strings.TrimPrefix(value, "@"))
	}
	if !json.Valid([]byte(value)) {
		return nil, fmt.Errorf("value is not valid JSON: %s", value)
	}
	return json.RawMessage(value), nil
}

func readJSONFile(path string) (json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("file %s does not contain valid JSON", path)
	}
	return json.RawMessage(data), nil
}

func readJSONReader(r io.Reader) (json.RawMessage, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("stdin does not contain valid JSON")
	}
	return json.RawMessage(data), nil
}
