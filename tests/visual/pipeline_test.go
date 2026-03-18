package visual_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pupersosition/tui-testing-skills/internal/visual"
)

func runtimeMetadata() map[string]interface{} {
	return map[string]interface{}{
		"cols":             24,
		"rows":             5,
		"theme":            "light",
		"color_mode":       "256",
		"locale":           "en_US.UTF-8",
		"renderer_version": "builtin-terminal-rasterizer/1.0",
	}
}

func TestSnapshotWritesPNGAndMetadata(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, time.March, 18, 1, 30, 0, 0, time.UTC)
	runDir := filepath.Join(t.TempDir(), "run-a")
	pipeline, err := visual.New(visual.Config{
		RunDir: runDir,
		Now: func() time.Time {
			return fixedNow
		},
	})
	if err != nil {
		t.Fatalf("visual.New failed: %v", err)
	}

	result := pipeline.Snapshot(
		"session-1",
		"counter-home",
		"Counter: 1\nPress q to quit",
		runtimeMetadata(),
	)
	if !result.OK {
		t.Fatalf("snapshot failed: %+v", result)
	}

	snapshotPath := result.Data["snapshot_path"].(string)
	metadataPath := result.Data["metadata_path"].(string)

	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("snapshot path missing: %v", err)
	}
	metadataBlob, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataBlob, &metadata); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if metadata["session_id"] != "session-1" {
		t.Fatalf("unexpected session_id: %v", metadata["session_id"])
	}
	if metadata["checkpoint"] != "counter-home" {
		t.Fatalf("unexpected checkpoint: %v", metadata["checkpoint"])
	}
	if metadata["created_at"] != "2026-03-18T01:30:00Z" {
		t.Fatalf("unexpected created_at: %v", metadata["created_at"])
	}

	runtime, ok := metadata["runtime"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing runtime metadata: %+v", metadata["runtime"])
	}
	if runtime["cols"] != float64(24) || runtime["rows"] != float64(5) {
		t.Fatalf("unexpected cols/rows: %+v", runtime)
	}
	if runtime["renderer_version"] != "builtin-terminal-rasterizer/1.0" {
		t.Fatalf("unexpected renderer_version: %+v", runtime)
	}
}

func TestAssertVisualPassAndFailDiffBehavior(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	baselinePipeline, err := visual.New(visual.Config{RunDir: filepath.Join(tmp, "baseline-run")})
	if err != nil {
		t.Fatalf("visual.New baseline failed: %v", err)
	}
	baselineSnapshot := baselinePipeline.Snapshot(
		"baseline-session",
		"screen",
		"Counter: 1\nPress q to quit",
		runtimeMetadata(),
	)
	if !baselineSnapshot.OK {
		t.Fatalf("baseline snapshot failed: %+v", baselineSnapshot)
	}
	baselinePath := baselineSnapshot.Data["snapshot_path"].(string)

	testPipeline, err := visual.New(visual.Config{RunDir: filepath.Join(tmp, "test-run")})
	if err != nil {
		t.Fatalf("visual.New test failed: %v", err)
	}

	sameSnapshot := testPipeline.Snapshot(
		"test-session",
		"screen",
		"Counter: 1\nPress q to quit",
		runtimeMetadata(),
	)
	if !sameSnapshot.OK {
		t.Fatalf("same snapshot failed: %+v", sameSnapshot)
	}

	passing := testPipeline.AssertVisual("test-session", "screen", baselinePath, 0)
	if !passing.OK {
		t.Fatalf("passing assert failed: %+v", passing)
	}
	if passing.Data["passed"] != true {
		t.Fatalf("expected passing result, got: %+v", passing.Data)
	}
	if passing.Data["difference_ratio"] != float64(0) {
		t.Fatalf("expected zero difference, got: %v", passing.Data["difference_ratio"])
	}
	if passing.Data["diff_artifact"] != nil {
		t.Fatalf("expected nil diff artifact on pass, got: %v", passing.Data["diff_artifact"])
	}

	differentSnapshot := testPipeline.Snapshot(
		"test-session",
		"screen",
		"Counter: 9\nPress q to quit",
		runtimeMetadata(),
	)
	if !differentSnapshot.OK {
		t.Fatalf("different snapshot failed: %+v", differentSnapshot)
	}

	failing := testPipeline.AssertVisual("test-session", "screen", baselinePath, 0)
	if !failing.OK {
		t.Fatalf("failing assert call failed: %+v", failing)
	}
	if failing.Data["passed"] != false {
		t.Fatalf("expected failed visual assertion, got: %+v", failing.Data)
	}

	ratio, ok := failing.Data["difference_ratio"].(float64)
	if !ok || ratio <= 0 {
		t.Fatalf("expected positive difference ratio, got: %v", failing.Data["difference_ratio"])
	}

	diffArtifact, ok := failing.Data["diff_artifact"].(string)
	if !ok || strings.TrimSpace(diffArtifact) == "" {
		t.Fatalf("expected diff artifact path, got: %v", failing.Data["diff_artifact"])
	}
	if _, err := os.Stat(diffArtifact); err != nil {
		t.Fatalf("diff artifact missing: %v", err)
	}
}

type unavailableRenderer struct{}

func (unavailableRenderer) Encode(_ string, _ []string, _ int) error {
	return visual.ErrRendererUnavailable
}

func TestRecordReportsRendererUnavailable(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	pipeline, err := visual.New(visual.Config{
		RunDir:      filepath.Join(tmp, "run-record"),
		GIFRenderer: unavailableRenderer{},
	})
	if err != nil {
		t.Fatalf("visual.New failed: %v", err)
	}

	snapshot := pipeline.Snapshot("session-record", "frame-1", "Frame 1", runtimeMetadata())
	if !snapshot.OK {
		t.Fatalf("snapshot failed: %+v", snapshot)
	}
	snapshotPath := snapshot.Data["snapshot_path"].(string)
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("missing snapshot: %v", err)
	}

	result := pipeline.Record(
		"session-record",
		filepath.Join(tmp, "review.gif"),
		nil,
		250,
	)
	if result.OK {
		t.Fatalf("expected renderer-unavailable failure, got success: %+v", result)
	}
	if result.Error == nil || result.Error.Code != "renderer_unavailable" {
		t.Fatalf("unexpected error payload: %+v", result.Error)
	}
	if !strings.Contains(result.Error.Message, "GIF renderer is unavailable") {
		t.Fatalf("unexpected renderer message: %q", result.Error.Message)
	}
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("snapshot was unexpectedly removed: %v", err)
	}
}
