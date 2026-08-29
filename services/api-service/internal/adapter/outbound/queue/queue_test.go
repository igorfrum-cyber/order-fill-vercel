package queue

import (
	"encoding/json"
	"testing"

	"order-fill/services/api-service/internal/app/port"
)

func TestEncodeMessage(t *testing.T) {
	message := port.JobMessage{
		JobID:      "job-1",
		Type:       "order_fill",
		Stage:      port.StageProcess,
		Brand:      "north",
		OrderMonth: "2026-09",
		Inputs: []port.MessageFile{
			{Role: "source", Name: "source.xlsx", StorageKey: "jobs/job-1/inputs/0-source.xlsx"},
			{Role: "blank", Name: "blank.xlsx", StorageKey: "jobs/job-1/inputs/1-blank.xlsx"},
		},
		Edits: []port.MessageEdit{
			{Key: "row-1", Value: "12", Comment: "ручная правка"},
		},
	}

	payload, err := encodeMessage(message)
	if err != nil {
		t.Fatalf("encode message: %v", err)
	}

	want := `{"job_id":"job-1","type":"order_fill","stage":"process","brand":"north",` +
		`"order_month":"2026-09","inputs":[` +
		`{"role":"source","name":"source.xlsx","storage_key":"jobs/job-1/inputs/0-source.xlsx"},` +
		`{"role":"blank","name":"blank.xlsx","storage_key":"jobs/job-1/inputs/1-blank.xlsx"}],` +
		`"edits":[{"key":"row-1","value":"12","comment":"ручная правка"}]}`
	if string(payload) != want {
		t.Fatalf("message json mismatch:\n got %s\nwant %s", payload, want)
	}

	var decoded port.JobMessage
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode message: %v", err)
	}
	if decoded.JobID != message.JobID || decoded.Stage != message.Stage || len(decoded.Inputs) != 2 {
		t.Fatalf("decoded message mismatch: %+v", decoded)
	}
}

func TestEncodeMessageOmitsEmptyEdits(t *testing.T) {
	payload, err := encodeMessage(port.JobMessage{
		JobID:  "job-2",
		Type:   "north_merge",
		Stage:  port.StageFinalize,
		Brand:  "north",
		Inputs: []port.MessageFile{},
	})
	if err != nil {
		t.Fatalf("encode message: %v", err)
	}

	want := `{"job_id":"job-2","type":"north_merge","stage":"finalize","brand":"north",` +
		`"order_month":"","inputs":[]}`
	if string(payload) != want {
		t.Fatalf("message json mismatch:\n got %s\nwant %s", payload, want)
	}
}

func TestNewPublisherRejectsInvalidURL(t *testing.T) {
	if _, err := NewPublisher("not-a-redis-url", ""); err == nil {
		t.Fatal("expected an error for an invalid queue url")
	}
}

func TestNewPublisherDefaultsStreamName(t *testing.T) {
	publisher, err := NewPublisher("redis://localhost:6379/0", "  ")
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	defer func() { _ = publisher.Close() }()

	if publisher.stream != DefaultStreamName {
		t.Fatalf("stream: got %q want %q", publisher.stream, DefaultStreamName)
	}
}
