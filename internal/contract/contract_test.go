package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type fixtureSet map[string]json.RawMessage

func TestGoldenRequestsParseAndValidate(t *testing.T) {
	requests := loadFixtures(t, "requests.json")
	assertFixtureCoverage(t, requests)

	for commandName, payload := range requests {
		commandName := commandName
		payload := payload
		t.Run(commandName, func(t *testing.T) {
			t.Parallel()
			req, err := ParseRequest(payload)
			if err != nil {
				t.Fatalf("ParseRequest failed: %v", err)
			}
			if string(req.Command) != commandName {
				t.Fatalf("unexpected command: got %q want %q", req.Command, commandName)
			}
			if _, err := DecodeParams(req.Command, req.Params); err != nil {
				t.Fatalf("DecodeParams failed: %v", err)
			}
		})
	}
}

func TestGoldenResponsesParseAndValidate(t *testing.T) {
	responses := loadFixtures(t, "responses.json")
	assertFixtureCoverage(t, responses)

	for commandName, payload := range responses {
		commandName := commandName
		payload := payload
		t.Run(commandName, func(t *testing.T) {
			t.Parallel()
			result, err := ParseResult(payload)
			if err != nil {
				t.Fatalf("ParseResult failed: %v", err)
			}
			if !result.OK {
				t.Fatalf("expected ok=true response for %s", commandName)
			}
		})
	}
}

func TestRejectsUnknownCommand(t *testing.T) {
	payload := []byte(`{"version":"1.0.0","command":"noop","params":{}}`)
	if _, err := ParseRequest(payload); err == nil {
		t.Fatal("expected unknown command error")
	}
}

func loadFixtures(t *testing.T, fileName string) fixtureSet {
	t.Helper()

	path := filepath.Join("testdata", "golden", fileName)
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var fixtures fixtureSet
	if err := json.Unmarshal(blob, &fixtures); err != nil {
		t.Fatalf("parse fixture %s: %v", path, err)
	}
	return fixtures
}

func assertFixtureCoverage(t *testing.T, fixtures fixtureSet) {
	t.Helper()
	if len(fixtures) != len(AllCommands) {
		t.Fatalf("fixture count mismatch: got %d want %d", len(fixtures), len(AllCommands))
	}
	for _, command := range AllCommands {
		if _, ok := fixtures[string(command)]; !ok {
			t.Fatalf("missing fixture for command %q", command)
		}
	}
}
