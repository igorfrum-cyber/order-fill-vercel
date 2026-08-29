package queue

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"order-fill/services/document-service/internal/app/port"
)

func TestDefaultStreamName(t *testing.T) {
	// api-service pushes onto this exact list; renaming it silently breaks the
	// handover between the two services.
	if DefaultStreamName != "order-fill:jobs" {
		t.Fatalf("stream name: got %q", DefaultStreamName)
	}
}

func TestDecodeMessageFromAPIService(t *testing.T) {
	payload := []byte(`{
		"job_id": "job-123",
		"type": "order_fill",
		"stage": "finalize",
		"brand": "ANGIOPHARM",
		"order_month": "2026-09",
		"inputs": [
			{"role": "source", "name": "order.xlsx", "storage_key": "jobs/job-123/inputs/order.xlsx"},
			{"role": "blank", "name": "blank.xlsx", "storage_key": "jobs/job-123/inputs/blank.xlsx"}
		],
		"edits": [
			{"key": "blank-1:row:12", "value": "10", "comment": "по договоренности"},
			{"key": "blank-1:row:13", "value": "0"}
		]
	}`)

	message, err := decodeMessage(payload)
	if err != nil {
		t.Fatalf("decode message: %v", err)
	}

	if message.JobID != "job-123" {
		t.Fatalf("job id: got %q", message.JobID)
	}
	if message.Type != "order_fill" {
		t.Fatalf("type: got %q", message.Type)
	}
	if message.Stage != port.StageFinalize {
		t.Fatalf("stage: got %q want %q", message.Stage, port.StageFinalize)
	}
	if message.Brand != "ANGIOPHARM" {
		t.Fatalf("brand: got %q", message.Brand)
	}
	if message.OrderMonth != "2026-09" {
		t.Fatalf("order month: got %q", message.OrderMonth)
	}

	if len(message.Inputs) != 2 {
		t.Fatalf("inputs: got %d want 2", len(message.Inputs))
	}
	wantInputs := []port.MessageFile{
		{Role: port.RoleSource, Name: "order.xlsx", StorageKey: "jobs/job-123/inputs/order.xlsx"},
		{Role: port.RoleBlank, Name: "blank.xlsx", StorageKey: "jobs/job-123/inputs/blank.xlsx"},
	}
	for index, want := range wantInputs {
		if message.Inputs[index] != want {
			t.Fatalf("input %d: got %+v want %+v", index, message.Inputs[index], want)
		}
	}

	wantEdits := []port.MessageEdit{
		{Key: "blank-1:row:12", Value: "10", Comment: "по договоренности"},
		{Key: "blank-1:row:13", Value: "0"},
	}
	if len(message.Edits) != len(wantEdits) {
		t.Fatalf("edits: got %d want %d", len(message.Edits), len(wantEdits))
	}
	for index, want := range wantEdits {
		if message.Edits[index] != want {
			t.Fatalf("edit %d: got %+v want %+v", index, message.Edits[index], want)
		}
	}
}

func TestDecodeMessageDefaultsStageToEmpty(t *testing.T) {
	// A process-stage message from api-service may omit the field; the use case
	// treats an empty stage as StageProcess.
	message, err := decodeMessage([]byte(`{"job_id":"job-1","type":"order_fill","inputs":[]}`))
	if err != nil {
		t.Fatalf("decode message: %v", err)
	}
	if message.Stage != "" {
		t.Fatalf("stage: got %q want empty", message.Stage)
	}
	if len(message.Edits) != 0 {
		t.Fatalf("edits: got %d want 0", len(message.Edits))
	}
}

func TestDecodeMessageRejectsBrokenPayload(t *testing.T) {
	if _, err := decodeMessage([]byte(`{"job_id":`)); err == nil {
		t.Fatal("expected an error for a truncated payload")
	}
}

func TestNewConsumerRejectsBadURL(t *testing.T) {
	if _, err := NewConsumer("not-a-url", "", discardLogger()); err == nil {
		t.Fatal("expected an error for an invalid queue url")
	}
}

func TestNewConsumerFallsBackToDefaultStream(t *testing.T) {
	consumer, err := NewConsumer("redis://localhost:6379/0", "  ", discardLogger())
	if err != nil {
		t.Fatalf("new consumer: %v", err)
	}
	defer func() { _ = consumer.Close() }()

	if consumer.stream != DefaultStreamName {
		t.Fatalf("stream: got %q want %q", consumer.stream, DefaultStreamName)
	}
}

func TestRunRequiresHandler(t *testing.T) {
	consumer, err := NewConsumer("redis://localhost:6379/0", "", discardLogger())
	if err != nil {
		t.Fatalf("new consumer: %v", err)
	}
	defer func() { _ = consumer.Close() }()

	if err := consumer.Run(context.Background(), nil); err == nil {
		t.Fatal("expected an error for a nil handler")
	}
}

// A cancelled context is a graceful shutdown, so Run returns nil without ever
// reaching Redis.
func TestRunStopsOnCancelledContext(t *testing.T) {
	consumer, err := NewConsumer("redis://localhost:6379/0", "", discardLogger())
	if err != nil {
		t.Fatalf("new consumer: %v", err)
	}
	defer func() { _ = consumer.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	handled := false
	err = consumer.Run(ctx, func(context.Context, port.JobMessage) error {
		handled = true
		return nil
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if handled {
		t.Fatal("handler must not run for a cancelled context")
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
