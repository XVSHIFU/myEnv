package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// checkJSONKeys rejects ambiguous objects before struct decoding can silently
// merge duplicate fields. Lock keys use canonical lowercase ASCII names,
// including platform/tool map keys; values retain their original case/Unicode.
// Input is already bounded by ReadInput.
func checkJSONKeys(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var value func(int) error
	value = func(depth int) error {
		if depth > 64 {
			return fmt.Errorf("JSON nesting limit exceeded")
		}
		token, err := d.Token()
		if err != nil {
			return err
		}
		delim, container := token.(json.Delim)
		if !container {
			return nil
		}
		switch delim {
		case '{':
			seen := make(map[string]bool)
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok {
					return fmt.Errorf("JSON object key must be a string")
				}
				if name == "" {
					return fmt.Errorf("empty lock JSON key")
				}
				for _, c := range name {
					if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
						return fmt.Errorf("lock JSON key %q must use lowercase ASCII letters, digits, underscores or hyphens", name)
					}
				}
				if seen[name] {
					return fmt.Errorf("duplicate JSON key %q", name)
				}
				seen[name] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("expected one JSON value")
	}
	return nil
}
