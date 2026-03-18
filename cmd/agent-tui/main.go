package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pupersosition/tui-testing-skills/internal/contract"
)

type options struct {
	request       string
	requestFile   string
	repl          bool
	rootOutputDir string
}

func main() {
	dispatcher := NewDispatcher(nil, nil)
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, dispatcher))
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, dispatcher *Dispatcher) int {
	opts, err := parseOptions(args, stderr)
	if err != nil {
		return 2
	}

	if opts.repl {
		return runREPL(stdin, stdout, dispatcher)
	}

	payload, err := loadSingleRequest(opts)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", marshalResult(errorResult("", "INVALID_ARGS", err.Error())))
		return 2
	}

	result := dispatcher.HandleRaw(payload)
	_, _ = fmt.Fprintf(stdout, "%s\n", marshalResult(result))
	return 0
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	fs := flag.NewFlagSet("agent-tui", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var opts options
	fs.StringVar(&opts.request, "request", "", "JSON request string")
	fs.StringVar(&opts.requestFile, "request-file", "", "Path to JSON request file")
	fs.BoolVar(&opts.repl, "repl", false, "Read one JSON request per line from stdin")
	fs.StringVar(&opts.rootOutputDir, "root-output-dir", "", "Reserved for visual runtime wiring")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	return opts, nil
}

func loadSingleRequest(opts options) ([]byte, error) {
	if opts.request != "" {
		return []byte(opts.request), nil
	}
	if opts.requestFile != "" {
		blob, err := os.ReadFile(opts.requestFile)
		if err != nil {
			return nil, err
		}
		return blob, nil
	}
	return nil, errors.New("provide --request or --request-file, or use --repl")
}

func runREPL(stdin io.Reader, stdout io.Writer, dispatcher *Dispatcher) int {
	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var raw json.RawMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			_, _ = fmt.Fprintf(stdout, "%s\n", marshalResult(errorResult("", errorCodeInvalidJSON, err.Error())))
			continue
		}

		result := dispatcher.HandleRaw([]byte(line))
		_, _ = fmt.Fprintf(stdout, "%s\n", marshalResult(result))
	}

	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintf(stdout, "%s\n", marshalResult(errorResult("", errorCodeInternal, err.Error())))
		return 1
	}
	return 0
}

func marshalResult(result contract.Result) string {
	blob, err := json.Marshal(result)
	if err != nil {
		fallback := errorResult(result.SessionID, errorCodeInternal, err.Error())
		blob, _ = json.Marshal(fallback)
		return string(blob)
	}
	return string(blob)
}
