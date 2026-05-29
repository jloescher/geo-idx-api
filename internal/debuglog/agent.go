package debuglog

import (
	"encoding/json"
	"os"
	"time"
)

const logPath = "/Users/jonathanloescher/Code/quantyra-geoidx/idx-api/.cursor/debug-4f5bac.log"

// Agent writes one NDJSON debug line for the active debug session.
func Agent(hypothesisID, location, message string, data map[string]any) {
	payload := map[string]any{
		"sessionId":    "4f5bac",
		"runId":        "post-fix",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	line, err := json.Marshal(payload)
	if err != nil {
		return
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(line, '\n'))
}
