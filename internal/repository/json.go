package repository

import "encoding/json"

func jsonUnmarshal(data []byte, v any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}
