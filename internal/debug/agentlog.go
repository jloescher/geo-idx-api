package debug

import (
	"encoding/json"
	"os"
	"time"
)

// #region agent log
const defaultLogPath = "/Users/jonathanloescher/Code/quantyra-geoidx/idx-api/.cursor/debug-4f5bac.log"

// Log writes one NDJSON debug line for agent investigation (no secrets).
func Log(hypothesisID, location, message string, data map[string]any) {
	payload := map[string]any{
		"sessionId":    "4f5bac",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"timestamp":    time.Now().UnixMilli(),
	}
	if data != nil {
		payload["data"] = data
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	f, err := os.OpenFile(defaultLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
	_ = f.Close()
}

// #endregion
