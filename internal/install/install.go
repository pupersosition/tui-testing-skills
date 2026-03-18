package install

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultSkill      = "bubbletea-tui-visual-test"
	DefaultSourceRoot = "skills"
)

var supportedAgents = []string{"claude", "copilot", "codex", "opencode"}

type Options struct {
	Agent      string
	Skill      string
	SourceRoot string
	Dest       string
	Force      bool
	DryRun     bool
	RepoRoot   string
}

type Paths struct {
	SourceDir      string
	DestinationDir string
}

type Resolved struct {
	Options Options
	Paths   Paths
}

type Result struct {
	Paths            Paths
	ReplacedExisting bool
}

type UnknownAgentError struct {
	Agent string
}

func (e *UnknownAgentError) Error() string {
	return fmt.Sprintf("unsupported agent: %q (supported: %s)", e.Agent, strings.Join(supportedAgents, ", "))
}

type SourceNotFoundError struct {
	Path string
}

func (e *SourceNotFoundError) Error() string {
	return fmt.Sprintf("skill source not found: %s", e.Path)
}

type DestinationExistsError struct {
	Path string
}

func (e *DestinationExistsError) Error() string {
	return fmt.Sprintf("destination already exists: %s. Use --force to replace it.", e.Path)
}

func IsUserError(err error) bool {
	var unknownAgent *UnknownAgentError
	var sourceNotFound *SourceNotFoundError
	var destinationExists *DestinationExistsError
	return errors.As(err, &unknownAgent) ||
		errors.As(err, &sourceNotFound) ||
		errors.As(err, &destinationExists)
}

func Resolve(opts Options) (Resolved, error) {
	normalized, err := normalizeOptions(opts)
	if err != nil {
		return Resolved{}, err
	}

	sourceDir, err := filepath.Abs(filepath.Join(normalized.RepoRoot, normalized.SourceRoot, normalized.Skill))
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve source path: %w", err)
	}

	destinationDir, err := resolveDestination(normalized)
	if err != nil {
		return Resolved{}, err
	}

	if sourceDir == destinationDir {
		return Resolved{}, fmt.Errorf("source and destination resolve to the same path: %s", sourceDir)
	}

	return Resolved{
		Options: normalized,
		Paths: Paths{
			SourceDir:      sourceDir,
			DestinationDir: destinationDir,
		},
	}, nil
}

func Install(resolved Resolved) (Result, error) {
	sourceInfo, err := os.Stat(resolved.Paths.SourceDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{Paths: resolved.Paths}, &SourceNotFoundError{Path: resolved.Paths.SourceDir}
		}
		return Result{Paths: resolved.Paths}, fmt.Errorf("stat source: %w", err)
	}
	if !sourceInfo.IsDir() {
		return Result{Paths: resolved.Paths}, &SourceNotFoundError{Path: resolved.Paths.SourceDir}
	}

	result := Result{
		Paths: resolved.Paths,
	}

	if _, err := os.Stat(resolved.Paths.DestinationDir); err == nil {
		if !resolved.Options.Force {
			return result, &DestinationExistsError{Path: resolved.Paths.DestinationDir}
		}
		result.ReplacedExisting = true
		if !resolved.Options.DryRun {
			if err := os.RemoveAll(resolved.Paths.DestinationDir); err != nil {
				return result, fmt.Errorf("remove existing destination: %w", err)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return result, fmt.Errorf("stat destination: %w", err)
	}

	if resolved.Options.DryRun {
		return result, nil
	}

	if err := os.MkdirAll(filepath.Dir(resolved.Paths.DestinationDir), 0o755); err != nil {
		return result, fmt.Errorf("create destination parent: %w", err)
	}
	if err := copyDir(resolved.Paths.SourceDir, resolved.Paths.DestinationDir); err != nil {
		return result, err
	}
	return result, nil
}

func normalizeOptions(opts Options) (Options, error) {
	normalized := opts
	normalized.Agent = strings.ToLower(strings.TrimSpace(opts.Agent))
	if normalized.Agent == "" {
		return Options{}, errors.New("--agent is required")
	}

	normalized.Skill = strings.TrimSpace(opts.Skill)
	if normalized.Skill == "" {
		normalized.Skill = DefaultSkill
	}

	normalized.SourceRoot = strings.TrimSpace(opts.SourceRoot)
	if normalized.SourceRoot == "" {
		normalized.SourceRoot = DefaultSourceRoot
	}

	if strings.TrimSpace(normalized.RepoRoot) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return Options{}, fmt.Errorf("determine working directory: %w", err)
		}
		normalized.RepoRoot = wd
	}
	repoRoot, err := filepath.Abs(normalized.RepoRoot)
	if err != nil {
		return Options{}, fmt.Errorf("resolve repo root: %w", err)
	}
	normalized.RepoRoot = repoRoot

	return normalized, nil
}

func resolveDestination(opts Options) (string, error) {
	if strings.TrimSpace(opts.Dest) != "" {
		return absolutePathWithHome(opts.Dest)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	var destinationRoot string
	switch opts.Agent {
	case "claude":
		destinationRoot = filepath.Join(homeDir, ".claude", "skills")
	case "copilot":
		destinationRoot = filepath.Join(homeDir, ".config", "copilot", "skills")
	case "codex":
		destinationRoot = filepath.Join(homeDir, ".codex", "skills")
	case "opencode":
		destinationRoot = filepath.Join(homeDir, ".config", "opencode", "skills")
	default:
		return "", &UnknownAgentError{Agent: opts.Agent}
	}

	return filepath.Abs(filepath.Join(destinationRoot, opts.Skill))
}

func absolutePathWithHome(pathValue string) (string, error) {
	if pathValue == "~" || strings.HasPrefix(pathValue, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if pathValue == "~" {
			pathValue = homeDir
		} else {
			pathValue = filepath.Join(homeDir, strings.TrimPrefix(pathValue, "~/"))
		}
	}
	return filepath.Abs(pathValue)
}

func copyDir(sourceDir, destinationDir string) error {
	return filepath.WalkDir(sourceDir, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relativePath, err := filepath.Rel(sourceDir, currentPath)
		if err != nil {
			return fmt.Errorf("compute relative path: %w", err)
		}

		targetPath := destinationDir
		if relativePath != "." {
			targetPath = filepath.Join(destinationDir, relativePath)
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("read entry info: %w", err)
		}

		switch {
		case entry.IsDir():
			mode := info.Mode().Perm()
			if mode == 0 {
				mode = 0o755
			}
			if err := os.MkdirAll(targetPath, mode); err != nil {
				return fmt.Errorf("create directory %s: %w", targetPath, err)
			}
			return nil
		case info.Mode()&os.ModeSymlink != 0:
			linkTarget, err := os.Readlink(currentPath)
			if err != nil {
				return fmt.Errorf("read symlink %s: %w", currentPath, err)
			}
			if err := os.Symlink(linkTarget, targetPath); err != nil {
				return fmt.Errorf("create symlink %s: %w", targetPath, err)
			}
			return nil
		default:
			mode := info.Mode().Perm()
			if mode == 0 {
				mode = 0o644
			}
			if err := copyFile(currentPath, targetPath, mode); err != nil {
				return err
			}
			return nil
		}
	})
}

func copyFile(sourcePath, destinationPath string, mode fs.FileMode) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", sourcePath, err)
	}
	defer sourceFile.Close()

	destinationFile, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("open destination file %s: %w", destinationPath, err)
	}
	defer destinationFile.Close()

	if _, err := io.Copy(destinationFile, sourceFile); err != nil {
		return fmt.Errorf("copy file %s -> %s: %w", sourcePath, destinationPath, err)
	}

	return nil
}
