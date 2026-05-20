package queue

import "encoding/json"

// laravelEnvelope is the Laravel database queue payload shape (Illuminate\Queue\CallQueuedHandler).
type laravelEnvelope struct {
	DisplayName string `json:"displayName"`
	Job         string `json:"job"`
	Data        struct {
		CommandName string `json:"commandName"`
	} `json:"data"`
}

// LegacyLaravelJobName returns the Laravel job class name when payload is a pre-cutover PHP job.
func LegacyLaravelJobName(raw []byte) string {
	var env laravelEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return ""
	}
	if env.Job == "" && env.DisplayName == "" {
		return ""
	}
	if env.DisplayName != "" {
		return env.DisplayName
	}
	return env.Data.CommandName
}

// IsLegacyLaravelPayload reports whether raw is a Laravel serialized queue job (not Go {"type":"..."}).
func IsLegacyLaravelPayload(raw []byte) bool {
	var p Payload
	if json.Unmarshal(raw, &p) == nil && p.Type != "" {
		return false
	}
	return LegacyLaravelJobName(raw) != ""
}
