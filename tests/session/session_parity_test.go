package session_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pupersosition/tui-testing-skills/internal/contract"
	"github.com/pupersosition/tui-testing-skills/internal/session"
)

type goldenResponses map[string]map[string]interface{}

func TestFixtureWorkflowParity(t *testing.T) {
	t.Parallel()

	mgr := session.NewManager()
	t.Cleanup(mgr.Shutdown)

	params := contract.OpenParams{
		Cmd:  "go run .",
		Cwd:  fixtureDir(t),
		Cols: 80,
		Rows: 24,
		Env: map[string]string{
			"TERM":   "xterm-256color",
			"TZ":     "UTC",
			"LANG":   "C.UTF-8",
			"LC_ALL": "C.UTF-8",
		},
		Locale:    "C.UTF-8",
		Theme:     "light",
		ColorMode: "256",
	}

	openRes := mgr.Open(params)
	assertOKResult(t, "open", openRes)
	sessionID := openRes.SessionID
	t.Cleanup(func() {
		_ = mgr.Close(contract.SessionParams{SessionID: sessionID})
	})

	waitReady := mgr.Wait(contract.WaitParams{
		SessionID: sessionID,
		MatchText: "STATUS: READY",
		TimeoutMS: 35000,
	})
	assertOKResult(t, "wait-ready", waitReady)

	pressRes := mgr.Press(contract.PressParams{
		SessionID: sessionID,
		Key:       "+",
	})
	assertOKResult(t, "press", pressRes)

	waitCount := mgr.Wait(contract.WaitParams{
		SessionID: sessionID,
		MatchRE:   `Counter:\s+1`,
		TimeoutMS: 10000,
	})
	assertOKResult(t, "wait-counter", waitCount)

	closeRes := mgr.Close(contract.SessionParams{SessionID: sessionID})
	assertOKResult(t, "close", closeRes)
}

func TestSessionResponsesMatchGoldenShape(t *testing.T) {
	mgr := session.NewManager(session.WithIDFactory(func() string { return "shape" }))
	t.Cleanup(mgr.Shutdown)

	golden := loadGoldenResponses(t)

	openRes := mgr.Open(contract.OpenParams{
		Cmd:       "cat",
		Cwd:       repoRoot(t),
		Cols:      80,
		Rows:      24,
		Locale:    "C.UTF-8",
		Theme:     "light",
		ColorMode: "256",
	})
	assertOKResult(t, "open", openRes)
	sessionID := openRes.SessionID
	t.Cleanup(func() {
		_ = mgr.Close(contract.SessionParams{SessionID: sessionID})
	})

	typeRes := mgr.Type(contract.TypeParams{
		SessionID: sessionID,
		Text:      "hello",
	})
	assertOKResult(t, "type", typeRes)

	pressRes := mgr.Press(contract.PressParams{
		SessionID: sessionID,
		Key:       "enter",
	})
	assertOKResult(t, "press", pressRes)

	waitRes := mgr.Wait(contract.WaitParams{
		SessionID: sessionID,
		MatchText: "hello",
		TimeoutMS: 5000,
	})
	assertOKResult(t, "wait", waitRes)

	closeRes := mgr.Close(contract.SessionParams{SessionID: sessionID})
	assertOKResult(t, "close", closeRes)

	assertHasGoldenDataKeys(t, "open", golden["open"], openRes)
	assertHasGoldenDataKeys(t, "type", golden["type"], typeRes)
	assertHasGoldenDataKeys(t, "press", golden["press"], pressRes)
	assertHasGoldenDataKeys(t, "wait", golden["wait"], waitRes)
	assertHasGoldenDataKeys(t, "close", golden["close"], closeRes)

	if matched, _ := waitRes.Data["matched"].(bool); !matched {
		t.Fatalf("wait response matched=false, want true")
	}
}

func TestExecuteAndIntegrationHooks(t *testing.T) {
	mgr := session.NewManager(session.WithIDFactory(func() string { return "hooks" }))
	t.Cleanup(mgr.Shutdown)

	openRaw, err := json.Marshal(contract.OpenParams{
		Cmd:       "cat",
		Cwd:       repoRoot(t),
		Cols:      80,
		Rows:      24,
		Locale:    "C.UTF-8",
		Theme:     "light",
		ColorMode: "256",
	})
	if err != nil {
		t.Fatalf("marshal open params: %v", err)
	}

	openRes := mgr.Execute(contract.CommandOpen, openRaw)
	assertOKResult(t, "execute-open", openRes)
	sessionID := openRes.SessionID
	t.Cleanup(func() {
		_ = mgr.Close(contract.SessionParams{SessionID: sessionID})
	})

	meta, ok := mgr.RuntimeMetadata(sessionID)
	if !ok {
		t.Fatalf("RuntimeMetadata returned missing session")
	}
	if got, _ := meta["locale"].(string); got != "C.UTF-8" {
		t.Fatalf("metadata locale = %q, want C.UTF-8", got)
	}
	if got, _ := meta["theme"].(string); got != "light" {
		t.Fatalf("metadata theme = %q, want light", got)
	}
	if got, _ := meta["color_mode"].(string); got != "256" {
		t.Fatalf("metadata color_mode = %q, want 256", got)
	}

	typeRaw, err := json.Marshal(contract.TypeParams{
		SessionID: sessionID,
		Text:      "hook-check",
	})
	if err != nil {
		t.Fatalf("marshal type params: %v", err)
	}
	assertOKResult(t, "execute-type", mgr.Execute(contract.CommandType, typeRaw))

	pressRaw, err := json.Marshal(contract.PressParams{
		SessionID: sessionID,
		Key:       "enter",
	})
	if err != nil {
		t.Fatalf("marshal press params: %v", err)
	}
	assertOKResult(t, "execute-press", mgr.Execute(contract.CommandPress, pressRaw))

	waitRaw, err := json.Marshal(contract.WaitParams{
		SessionID: sessionID,
		MatchText: "hook-check",
		TimeoutMS: 4000,
	})
	if err != nil {
		t.Fatalf("marshal wait params: %v", err)
	}
	assertOKResult(t, "execute-wait", mgr.Execute(contract.CommandWait, waitRaw))

	screen, ok := mgr.ScreenText(sessionID)
	if !ok {
		t.Fatalf("ScreenText returned missing session")
	}
	if !strings.Contains(screen, "hook-check") {
		t.Fatalf("screen text %q does not include typed marker", screen)
	}

	ids := mgr.ActiveSessionIDs()
	if len(ids) != 1 || ids[0] != sessionID {
		t.Fatalf("ActiveSessionIDs = %v, want [%s]", ids, sessionID)
	}

	closeRaw, err := json.Marshal(contract.SessionParams{SessionID: sessionID})
	if err != nil {
		t.Fatalf("marshal close params: %v", err)
	}
	assertOKResult(t, "execute-close", mgr.Execute(contract.CommandClose, closeRaw))
}

func assertOKResult(t *testing.T, name string, result contract.Result) {
	t.Helper()
	if !result.OK {
		t.Fatalf("%s failed: %+v", name, result)
	}
	assertContractResult(t, name, result)
}

func assertContractResult(t *testing.T, name string, result contract.Result) {
	t.Helper()
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("%s marshal result: %v", name, err)
	}
	if _, err := contract.ParseResult(raw); err != nil {
		t.Fatalf("%s ParseResult: %v", name, err)
	}
}

func assertHasGoldenDataKeys(t *testing.T, command string, golden map[string]interface{}, result contract.Result) {
	t.Helper()

	if command == "" {
		t.Fatalf("command must be set")
	}
	if golden == nil {
		t.Fatalf("missing golden entry for %q", command)
	}
	if goldenOK, _ := golden["ok"].(bool); !goldenOK {
		t.Fatalf("golden %q should be ok=true", command)
	}
	if result.SessionID == "" {
		t.Fatalf("%s session_id empty", command)
	}

	goldenData, _ := golden["data"].(map[string]interface{})
	for key := range goldenData {
		if _, ok := result.Data[key]; !ok {
			t.Fatalf("%s result missing golden data key %q", command, key)
		}
	}
}

func loadGoldenResponses(t *testing.T) goldenResponses {
	t.Helper()
	path := filepath.Join(repoRoot(t), "internal", "contract", "testdata", "golden", "responses.json")
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden responses: %v", err)
	}

	var parsed goldenResponses
	if err := json.Unmarshal(blob, &parsed); err != nil {
		t.Fatalf("parse golden responses: %v", err)
	}
	return parsed
}

func fixtureDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "skills", "bubbletea-tui-visual-test", "assets", "fixtures", "bubbletea-counter")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	// tests/session -> repository root
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

func init() {
	// Keep tests deterministic on slower runners.
	time.Local = time.UTC
}
