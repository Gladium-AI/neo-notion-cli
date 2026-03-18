package notion

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// Select applies a jq-like field path expression to JSON data.
func Select(data []byte, expr string) ([]byte, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("parse select expression: %w", err)
	}

	var input interface{}
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("unmarshal for select: %w", err)
	}

	iter := query.Run(input)
	var results []interface{}
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("select error: %w", err)
		}
		results = append(results, v)
	}

	if len(results) == 1 {
		return json.Marshal(results[0])
	}
	return json.Marshal(results)
}
