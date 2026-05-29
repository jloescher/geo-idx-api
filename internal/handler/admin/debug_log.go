package admin

import (
	"github.com/quantyralabs/idx-api/internal/debuglog"
)

func agentDebugLog(hypothesisID, location, message string, data map[string]any) {
	debuglog.Agent(hypothesisID, location, message, data)
}
