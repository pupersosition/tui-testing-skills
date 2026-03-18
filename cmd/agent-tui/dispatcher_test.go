package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pupersosition/tui-testing-skills/internal/contract"
)

func TestDispatcherSuccessEnvelopeParity(t *testing.T) {
	fixtures := loadRequestFixtures(t)
	dispatcher := NewDispatcher(successSessionRuntime{}, successVisualRuntime{})

	for _, command := range contract.AllCommands {
		command := command
		payload := fixtures[string(command)]
		t.Run(string(command), func(t *testing.T) {
			t.Parallel()
			result := dispatcher.HandleRaw(payload)
			if !result.OK {
				t.Fatalf("expected ok=true, got error: %+v", result.Error)
			}
			assertContractResult(t, result)
		})
	}
}

func TestDispatcherFailureEnvelopeParity(t *testing.T) {
	fixtures := loadRequestFixtures(t)
	dispatcher := NewDispatcher(failingSessionRuntime{}, failingVisualRuntime{})

	for _, command := range contract.AllCommands {
		command := command
		payload := fixtures[string(command)]
		t.Run(string(command), func(t *testing.T) {
			t.Parallel()
			result := dispatcher.HandleRaw(payload)
			if result.OK {
				t.Fatalf("expected ok=false for %q", command)
			}
			if result.Error == nil {
				t.Fatalf("expected error payload for %q", command)
			}
			assertContractResult(t, result)
		})
	}
}

func TestDispatcherInvalidRequestReturnsContractFailure(t *testing.T) {
	dispatcher := NewDispatcher(successSessionRuntime{}, successVisualRuntime{})
	result := dispatcher.HandleRaw([]byte(`{"version":"1.0.0","command":"open","params":[]}`))
	if result.OK {
		t.Fatal("expected request validation failure")
	}
	if result.Error == nil || result.Error.Code != errorCodeInvalidRequest {
		t.Fatalf("unexpected error payload: %+v", result.Error)
	}
	assertContractResult(t, result)
}

func TestDefaultRuntimesReturnNotImplemented(t *testing.T) {
	fixtures := loadRequestFixtures(t)
	dispatcher := NewDispatcher(nil, nil)

	for _, command := range contract.AllCommands {
		result := dispatcher.HandleRaw(fixtures[string(command)])
		if result.OK {
			t.Fatalf("expected default runtime to reject %q", command)
		}
		if result.Error == nil || result.Error.Code != errorCodeNotImplemented {
			t.Fatalf("expected NOT_IMPLEMENTED for %q, got %+v", command, result.Error)
		}
		assertContractResult(t, result)
	}
}

type requestFixtures map[string]json.RawMessage

func loadRequestFixtures(t *testing.T) requestFixtures {
	t.Helper()
	path := filepath.Join("..", "..", "internal", "contract", "testdata", "golden", "requests.json")
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}
	var fixtures requestFixtures
	if err := json.Unmarshal(blob, &fixtures); err != nil {
		t.Fatalf("parse fixtures: %v", err)
	}
	return fixtures
}

func assertContractResult(t *testing.T, result contract.Result) {
	t.Helper()
	blob, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if _, err := contract.ParseResult(blob); err != nil {
		t.Fatalf("result does not satisfy contract: %v", err)
	}
}

type successSessionRuntime struct{}

func (successSessionRuntime) Open(params contract.OpenParams) contract.Result {
	return contract.Result{OK: true, SessionID: "session-open", Data: map[string]interface{}{"command": "open"}}
}

func (successSessionRuntime) Close(params contract.SessionParams) contract.Result {
	return contract.Result{OK: true, SessionID: params.SessionID, Data: map[string]interface{}{"command": "close"}}
}

func (successSessionRuntime) Press(params contract.PressParams) contract.Result {
	return contract.Result{OK: true, SessionID: params.SessionID, Data: map[string]interface{}{"command": "press"}}
}

func (successSessionRuntime) Type(params contract.TypeParams) contract.Result {
	return contract.Result{OK: true, SessionID: params.SessionID, Data: map[string]interface{}{"command": "type"}}
}

func (successSessionRuntime) Wait(params contract.WaitParams) contract.Result {
	return contract.Result{OK: true, SessionID: params.SessionID, Data: map[string]interface{}{"command": "wait"}}
}

type successVisualRuntime struct{}

func (successVisualRuntime) Snapshot(params contract.SnapshotParams) contract.Result {
	return contract.Result{OK: true, SessionID: params.SessionID, Data: map[string]interface{}{"command": "snapshot"}}
}

func (successVisualRuntime) AssertVisual(params contract.AssertVisualParams) contract.Result {
	return contract.Result{OK: true, SessionID: params.SessionID, Data: map[string]interface{}{"command": "assert-visual"}}
}

func (successVisualRuntime) Record(params contract.RecordParams) contract.Result {
	return contract.Result{OK: true, SessionID: params.SessionID, Data: map[string]interface{}{"command": "record"}}
}

type failingSessionRuntime struct{}

func (failingSessionRuntime) Open(params contract.OpenParams) contract.Result {
	return errorResult("session-open", "OPEN_FAILED", "open failed in test runtime")
}

func (failingSessionRuntime) Close(params contract.SessionParams) contract.Result {
	return errorResult(params.SessionID, "CLOSE_FAILED", "close failed in test runtime")
}

func (failingSessionRuntime) Press(params contract.PressParams) contract.Result {
	return errorResult(params.SessionID, "INTERACTION_FAILED", "press failed in test runtime")
}

func (failingSessionRuntime) Type(params contract.TypeParams) contract.Result {
	return errorResult(params.SessionID, "INTERACTION_FAILED", "type failed in test runtime")
}

func (failingSessionRuntime) Wait(params contract.WaitParams) contract.Result {
	return errorResult(params.SessionID, "WAIT_FAILED", "wait failed in test runtime")
}

type failingVisualRuntime struct{}

func (failingVisualRuntime) Snapshot(params contract.SnapshotParams) contract.Result {
	return errorResult(params.SessionID, "SNAPSHOT_FAILED", "snapshot failed in test runtime")
}

func (failingVisualRuntime) AssertVisual(params contract.AssertVisualParams) contract.Result {
	return errorResult(params.SessionID, "ASSERT_VISUAL_FAILED", "assert-visual failed in test runtime")
}

func (failingVisualRuntime) Record(params contract.RecordParams) contract.Result {
	return errorResult(params.SessionID, "RECORD_FAILED", "record failed in test runtime")
}
