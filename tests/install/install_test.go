package install_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	installer "github.com/pupersosition/tui-testing-skills/internal/install"
)

func TestInstallSupportedAgentSuccess(t *testing.T) {
	f := newFixture(t)

	resolved, err := installer.Resolve(installer.Options{
		Agent:    "codex",
		RepoRoot: f.repoRoot,
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	result, err := installer.Install(resolved)
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	wantDestination := filepath.Join(f.homeDir, ".codex", "skills", installer.DefaultSkill)
	if result.Paths.DestinationDir != wantDestination {
		t.Fatalf("destination mismatch: got %q want %q", result.Paths.DestinationDir, wantDestination)
	}

	assertFileContent(t, filepath.Join(wantDestination, "SKILL.md"), "initial")
}

func TestUnknownAgentRejected(t *testing.T) {
	f := newFixture(t)

	_, err := installer.Resolve(installer.Options{
		Agent:    "unknown",
		RepoRoot: f.repoRoot,
	})
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}

	var unknownAgentErr *installer.UnknownAgentError
	if !errors.As(err, &unknownAgentErr) {
		t.Fatalf("expected UnknownAgentError, got %T: %v", err, err)
	}
}

func TestDestinationOverride(t *testing.T) {
	f := newFixture(t)
	customDestination := filepath.Join(f.repoRoot, "custom", "skill-install")

	resolved, err := installer.Resolve(installer.Options{
		Agent:    "claude",
		RepoRoot: f.repoRoot,
		Dest:     customDestination,
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	result, err := installer.Install(resolved)
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	if result.Paths.DestinationDir != customDestination {
		t.Fatalf("destination mismatch: got %q want %q", result.Paths.DestinationDir, customDestination)
	}

	assertFileContent(t, filepath.Join(customDestination, "SKILL.md"), "initial")
	defaultDestination := filepath.Join(f.homeDir, ".claude", "skills", installer.DefaultSkill)
	assertNotExists(t, defaultDestination)
}

func TestOverwriteGuardrails(t *testing.T) {
	f := newFixture(t)
	destination := filepath.Join(f.repoRoot, "dest", "skill")

	initialResolved, err := installer.Resolve(installer.Options{
		Agent:    "codex",
		RepoRoot: f.repoRoot,
		Dest:     destination,
	})
	if err != nil {
		t.Fatalf("resolve initial install: %v", err)
	}

	if _, err := installer.Install(initialResolved); err != nil {
		t.Fatalf("install initial: %v", err)
	}

	writeSourceSkill(t, f.repoRoot, "updated")

	rejectResolved, err := installer.Resolve(installer.Options{
		Agent:    "codex",
		RepoRoot: f.repoRoot,
		Dest:     destination,
	})
	if err != nil {
		t.Fatalf("resolve overwrite rejection: %v", err)
	}

	if _, err := installer.Install(rejectResolved); err == nil {
		t.Fatal("expected destination exists error")
	} else {
		var destinationExistsErr *installer.DestinationExistsError
		if !errors.As(err, &destinationExistsErr) {
			t.Fatalf("expected DestinationExistsError, got %T: %v", err, err)
		}
	}

	forceResolved, err := installer.Resolve(installer.Options{
		Agent:    "codex",
		RepoRoot: f.repoRoot,
		Dest:     destination,
		Force:    true,
	})
	if err != nil {
		t.Fatalf("resolve force overwrite: %v", err)
	}

	result, err := installer.Install(forceResolved)
	if err != nil {
		t.Fatalf("force install: %v", err)
	}
	if !result.ReplacedExisting {
		t.Fatal("expected ReplacedExisting=true for force overwrite")
	}

	assertFileContent(t, filepath.Join(destination, "SKILL.md"), "updated")
}

func TestDryRunReportsWithoutWriting(t *testing.T) {
	f := newFixture(t)
	destination := filepath.Join(f.repoRoot, "dest", "dry-run")

	resolved, err := installer.Resolve(installer.Options{
		Agent:    "opencode",
		RepoRoot: f.repoRoot,
		Dest:     destination,
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	result, err := installer.Install(resolved)
	if err != nil {
		t.Fatalf("dry-run install: %v", err)
	}
	if result.ReplacedExisting {
		t.Fatal("expected ReplacedExisting=false for new dry-run destination")
	}

	assertNotExists(t, destination)
}

type fixture struct {
	repoRoot string
	homeDir  string
}

func newFixture(t *testing.T) fixture {
	t.Helper()

	repoRoot := t.TempDir()
	homeDir := filepath.Join(repoRoot, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("create home dir: %v", err)
	}
	t.Setenv("HOME", homeDir)

	writeSourceSkill(t, repoRoot, "initial")

	return fixture{
		repoRoot: repoRoot,
		homeDir:  homeDir,
	}
}

func writeSourceSkill(t *testing.T, repoRoot, content string) {
	t.Helper()

	sourceDir := filepath.Join(repoRoot, installer.DefaultSourceRoot, installer.DefaultSkill)
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write source skill file: %v", err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("file content mismatch for %s: got %q want %q", path, string(data), want)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Fatalf("path should not exist: %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected stat error for %s: %v", path, err)
	}
}
