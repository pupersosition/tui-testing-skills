package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pupersosition/tui-testing-skills/internal/install"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	opts, err := parseArgs(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	resolved, err := install.Resolve(opts)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Agent: %s\n", resolved.Options.Agent)
	fmt.Fprintf(stdout, "Skill: %s\n", resolved.Options.Skill)
	fmt.Fprintf(stdout, "Source: %s\n", resolved.Paths.SourceDir)
	fmt.Fprintf(stdout, "Destination: %s\n", resolved.Paths.DestinationDir)

	result, err := install.Install(resolved)
	if err != nil {
		if install.IsUserError(err) {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Fprintf(stderr, "Unexpected error: %v\n", err)
		return 2
	}

	if resolved.Options.DryRun {
		if result.ReplacedExisting {
			fmt.Fprintf(stdout, "[dry-run] Would remove existing destination: %s\n", resolved.Paths.DestinationDir)
		}
		fmt.Fprintf(stdout, "[dry-run] Would copy: %s -> %s\n", resolved.Paths.SourceDir, resolved.Paths.DestinationDir)
		return 0
	}

	fmt.Fprintf(stdout, "Installed skill to: %s\n", resolved.Paths.DestinationDir)
	return 0
}

func parseArgs(args []string, stderr io.Writer) (install.Options, error) {
	flagSet := flag.NewFlagSet("install-skill", flag.ContinueOnError)
	flagSet.SetOutput(stderr)

	opts := install.Options{}
	flagSet.StringVar(&opts.Agent, "agent", "", "Target agent (required): claude|copilot|codex|opencode")
	flagSet.StringVar(&opts.Skill, "skill", install.DefaultSkill, "Skill folder name under the source root")
	flagSet.StringVar(&opts.SourceRoot, "source-root", install.DefaultSourceRoot, "Root directory containing reusable skills")
	flagSet.StringVar(&opts.Dest, "dest", "", "Explicit destination directory for the skill")
	flagSet.BoolVar(&opts.Force, "force", false, "Replace destination if it already exists")
	flagSet.BoolVar(&opts.DryRun, "dry-run", false, "Show planned actions without writing files")

	flagSet.Usage = func() {
		fmt.Fprintf(flagSet.Output(), "Usage: %s --agent <agent> [options]\n", flagSet.Name())
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(args); err != nil {
		return install.Options{}, err
	}
	if flagSet.NArg() != 0 {
		return install.Options{}, fmt.Errorf("unexpected positional arguments: %s", strings.Join(flagSet.Args(), " "))
	}
	if strings.TrimSpace(opts.Agent) == "" {
		return install.Options{}, errors.New("--agent is required")
	}

	return opts, nil
}
