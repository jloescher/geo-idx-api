//go:build smoke

package smoke

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func jsonStringAt(body []byte, path string) (string, bool) {
	v, ok := jsonValueAt(body, path)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func jsonValueAt(body []byte, path string) (any, bool) {
	if path == "" {
		var v any
		if err := json.Unmarshal(body, &v); err != nil {
			return nil, false
		}
		return v, true
	}
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, false
	}
	cur := root
	for _, part := range strings.Split(path, ".") {
		switch node := cur.(type) {
		case map[string]any:
			v, ok := node[part]
			if !ok {
				return nil, false
			}
			cur = v
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(node) {
				return nil, false
			}
			cur = node[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

func checkJSONExpectations(body []byte, expects map[string]string) []string {
	var failures []string
	for path, wantType := range expects {
		val, ok := jsonValueAt(body, path)
		if !ok {
			failures = append(failures, fmt.Sprintf("json path %q missing", path))
			continue
		}
		if err := assertJSONType(path, val, wantType); err != nil {
			failures = append(failures, err.Error())
		}
	}
	return failures
}

func checkJSONEquals(body []byte, expects map[string]any) []string {
	var failures []string
	for path, want := range expects {
		got, ok := jsonValueAt(body, path)
		if !ok {
			failures = append(failures, fmt.Sprintf("json path %q missing (want %v)", path, want))
			continue
		}
		if !jsonValuesEqual(got, want) {
			failures = append(failures, fmt.Sprintf("json path %q: got %v, want %v", path, got, want))
		}
	}
	return failures
}

func assertJSONType(path string, val any, wantType string) error {
	switch wantType {
	case "any":
		return nil
	case "string":
		if _, ok := val.(string); !ok {
			return fmt.Errorf("json path %q: want string, got %T", path, val)
		}
	case "number":
		switch val.(type) {
		case float64, json.Number:
		default:
			return fmt.Errorf("json path %q: want number, got %T", path, val)
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("json path %q: want boolean, got %T", path, val)
		}
	case "array":
		if _, ok := val.([]any); !ok {
			return fmt.Errorf("json path %q: want array, got %T", path, val)
		}
	case "object":
		if _, ok := val.(map[string]any); !ok {
			return fmt.Errorf("json path %q: want object, got %T", path, val)
		}
	default:
		return fmt.Errorf("json path %q: unknown type %q", path, wantType)
	}
	return nil
}

func jsonValuesEqual(got, want any) bool {
	switch w := want.(type) {
	case float64:
		g, ok := got.(float64)
		return ok && math.Abs(g-w) < 1e-9
	case int:
		g, ok := got.(float64)
		return ok && int(g) == w
	case string, bool:
		return got == want
	default:
		gb, _ := json.Marshal(got)
		wb, _ := json.Marshal(want)
		return string(gb) == string(wb)
	}
}

func isJSONArray(body []byte) bool {
	var arr []any
	return json.Unmarshal(body, &arr) == nil
}

func isJSONObject(body []byte) bool {
	var obj map[string]any
	return json.Unmarshal(body, &obj) == nil
}

func collectJSONPaths(body []byte, prefix string) map[string]any {
	out := map[string]any{}
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return out
	}
	walkJSONPaths(root, prefix, out)
	return out
}

func walkJSONPaths(v any, prefix string, out map[string]any) {
	switch node := v.(type) {
	case map[string]any:
		for k, child := range node {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			out[path] = child
			walkJSONPaths(child, path, out)
		}
	case []any:
		if prefix != "" {
			out[prefix] = node
		}
	}
}
