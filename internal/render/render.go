// Package render handles output formatting: JSON, YAML, raw, and pretty.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/paoloanzn/neo-notion-cli/internal/config"
)

// Output writes data to the configured destination in the configured format.
func Output(cfg *config.Config, data []byte) error {
	if cfg.Quiet {
		return nil
	}

	w := writer(cfg)

	switch cfg.OutputFormat {
	case "yaml":
		return outputYAML(w, data)
	case "raw":
		return outputRaw(w, data)
	case "pretty":
		return outputPretty(w, data)
	default: // json
		return outputJSON(w, data)
	}
}

func writer(cfg *config.Config) io.Writer {
	if cfg.OutputFile != "" {
		f, err := os.Create(cfg.OutputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening output file: %v\n", err)
			return os.Stdout
		}
		return f
	}
	return os.Stdout
}

func outputJSON(w io.Writer, data []byte) error {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		// Not valid JSON; output as-is.
		_, err := fmt.Fprintln(w, string(data))
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(obj)
}

func outputYAML(w io.Writer, data []byte) error {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		_, err := fmt.Fprintln(w, string(data))
		return err
	}
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	defer enc.Close()
	return enc.Encode(obj)
}

func outputRaw(w io.Writer, data []byte) error {
	_, err := w.Write(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w)
	return err
}

func outputPretty(w io.Writer, data []byte) error {
	// Pretty mode: indented JSON with colors would go here.
	// For now, same as JSON.
	return outputJSON(w, data)
}
