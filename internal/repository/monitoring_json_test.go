package repository

import (
	"encoding/json"
	"testing"
)

func TestQueueCountJSONUsesSnakeCase(t *testing.T) {
	raw, err := json.Marshal(QueueCount{
		Queue: "bridge-sync-fetch", Pending: 3, Reserved: 1, Failed: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"queue", "pending", "reserved", "failed"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("missing json key %q in %s", key, raw)
		}
	}
	if _, ok := m["Queue"]; ok {
		t.Fatalf("exported Go field name in JSON: %s", raw)
	}
}

func TestJobTypeCountJSONUsesSnakeCase(t *testing.T) {
	raw, err := json.Marshal(JobTypeCount{
		Queue: "default", JobType: "App\\Jobs\\SyncKickoff", Count: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"queue", "job_type", "count"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("missing json key %q in %s", key, raw)
		}
	}
}
