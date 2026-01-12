package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConformance_MessageJSONGolden(t *testing.T) {
	message := &a2av1.Message{
		MessageId: "msg-1",
		ContextId: "ctx-1",
		TaskId:    "task-1",
		Role:      a2av1.Role_ROLE_AGENT,
		Parts: []*a2av1.Part{
			{Part: &a2av1.Part_Text{Text: "hola"}},
		},
	}
	assertJSONGolden(t, "message.json", message)
}

func TestConformance_TaskJSONGolden(t *testing.T) {
	ts := timestamppb.New(time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC))
	message := &a2av1.Message{
		MessageId: "msg-1",
		ContextId: "ctx-1",
		TaskId:    "task-1",
		Role:      a2av1.Role_ROLE_AGENT,
		Parts: []*a2av1.Part{
			{Part: &a2av1.Part_Text{Text: "ok"}},
		},
	}
	task := &a2av1.Task{
		Id:        "task-1",
		ContextId: "ctx-1",
		Status: &a2av1.TaskStatus{
			State:     a2av1.TaskState_TASK_STATE_COMPLETED,
			Message:   message,
			Timestamp: ts,
		},
	}
	assertJSONGolden(t, "task.json", task)
}

func assertJSONGolden(t *testing.T, name string, msg proto.Message) {
	t.Helper()
	opts := protojson.MarshalOptions{Indent: "  "}
	payload, err := opts.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	path := filepath.Join("testdata", name)
	golden, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	got := normalizeJSON(t, string(payload))
	want := normalizeJSON(t, string(golden))
	if got != want {
		t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

func normalizeJSON(t *testing.T, raw string) string {
	t.Helper()
	raw = strings.TrimSpace(raw)
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	normalized, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return string(normalized)
}
