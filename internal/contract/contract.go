package contract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

const Version = "1.0.0"

type Command string

const (
	CommandOpen         Command = "open"
	CommandClose        Command = "close"
	CommandPress        Command = "press"
	CommandType         Command = "type"
	CommandWait         Command = "wait"
	CommandSnapshot     Command = "snapshot"
	CommandAssertVisual Command = "assert-visual"
	CommandRecord       Command = "record"
)

var AllCommands = []Command{
	CommandOpen,
	CommandClose,
	CommandPress,
	CommandType,
	CommandWait,
	CommandSnapshot,
	CommandAssertVisual,
	CommandRecord,
}

var validCommands = map[Command]struct{}{
	CommandOpen:         {},
	CommandClose:        {},
	CommandPress:        {},
	CommandType:         {},
	CommandWait:         {},
	CommandSnapshot:     {},
	CommandAssertVisual: {},
	CommandRecord:       {},
}

type Request struct {
	Version string          `json:"version"`
	Command Command         `json:"command"`
	Params  json.RawMessage `json:"params"`
}

type OpenParams struct {
	Cmd       string            `json:"cmd"`
	Cwd       string            `json:"cwd"`
	Cols      int               `json:"cols"`
	Rows      int               `json:"rows"`
	Env       map[string]string `json:"env,omitempty"`
	Locale    string            `json:"locale,omitempty"`
	Theme     string            `json:"theme,omitempty"`
	ColorMode string            `json:"color_mode,omitempty"`
}

type SessionParams struct {
	SessionID string `json:"session_id"`
}

type PressParams struct {
	SessionID string `json:"session_id"`
	Key       string `json:"key"`
}

type TypeParams struct {
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
}

type WaitParams struct {
	SessionID string `json:"session_id"`
	MatchText string `json:"match_text,omitempty"`
	MatchRE   string `json:"match_regex,omitempty"`
	TimeoutMS int    `json:"timeout_ms"`
}

type SnapshotParams struct {
	SessionID string `json:"session_id"`
	Name      string `json:"name"`
	OutputDir string `json:"output_dir,omitempty"`
}

type AssertVisualParams struct {
	SessionID    string  `json:"session_id"`
	Name         string  `json:"name"`
	BaselinePath string  `json:"baseline_path"`
	Threshold    float64 `json:"threshold,omitempty"`
}

type RecordParams struct {
	SessionID  string `json:"session_id"`
	OutputPath string `json:"output_path"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Result struct {
	OK        bool                   `json:"ok"`
	SessionID string                 `json:"session_id"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     *ErrorPayload          `json:"error,omitempty"`
}

func ParseRequest(raw []byte) (Request, error) {
	var req Request
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return Request{}, fmt.Errorf("decode request: %w", err)
	}
	if err := req.Validate(); err != nil {
		return Request{}, err
	}
	return req, nil
}

func ParseResult(raw []byte) (Result, error) {
	fields, err := decodeObject(raw)
	if err != nil {
		return Result{}, err
	}
	if err := requireFields(fields, "ok", "session_id"); err != nil {
		return Result{}, err
	}

	var out Result
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return Result{}, fmt.Errorf("decode result: %w", err)
	}
	if err := out.Validate(); err != nil {
		return Result{}, err
	}
	return out, nil
}

func (r Request) Validate() error {
	if r.Version != Version {
		return fmt.Errorf("unsupported version: %q", r.Version)
	}
	if _, ok := validCommands[r.Command]; !ok {
		return fmt.Errorf("unsupported command: %q", r.Command)
	}
	if len(r.Params) == 0 {
		return errors.New("params is required")
	}
	return validateParams(r.Command, r.Params)
}

func (r Result) Validate() error {
	if r.OK {
		if r.Error != nil {
			return errors.New("result.error must be absent when ok=true")
		}
		return nil
	}
	if r.Error == nil {
		return errors.New("result.error is required when ok=false")
	}
	if r.Error.Code == "" || r.Error.Message == "" {
		return errors.New("result.error.code and result.error.message are required")
	}
	return nil
}

func DecodeParams(command Command, raw json.RawMessage) (interface{}, error) {
	switch command {
	case CommandOpen:
		return decodeOpenParams(raw)
	case CommandClose:
		return decodeSessionParams(raw)
	case CommandPress:
		return decodePressParams(raw)
	case CommandType:
		return decodeTypeParams(raw)
	case CommandWait:
		return decodeWaitParams(raw)
	case CommandSnapshot:
		return decodeSnapshotParams(raw)
	case CommandAssertVisual:
		return decodeAssertVisualParams(raw)
	case CommandRecord:
		return decodeRecordParams(raw)
	default:
		return nil, fmt.Errorf("unsupported command: %q", command)
	}
}

func validateParams(command Command, raw json.RawMessage) error {
	_, err := DecodeParams(command, raw)
	return err
}

func decodeOpenParams(raw json.RawMessage) (OpenParams, error) {
	fields, err := decodeObject(raw)
	if err != nil {
		return OpenParams{}, err
	}
	if err := requireFields(fields, "cmd", "cwd", "cols", "rows"); err != nil {
		return OpenParams{}, err
	}

	var params OpenParams
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&params); err != nil {
		return OpenParams{}, fmt.Errorf("invalid open params: %w", err)
	}
	if params.Cmd == "" || params.Cwd == "" {
		return OpenParams{}, errors.New("open cmd and cwd must be non-empty")
	}
	if params.Cols < 20 || params.Rows < 10 {
		return OpenParams{}, errors.New("open cols must be >=20 and rows must be >=10")
	}
	if params.ColorMode != "" {
		switch params.ColorMode {
		case "16", "256", "truecolor":
		default:
			return OpenParams{}, fmt.Errorf("invalid color_mode: %q", params.ColorMode)
		}
	}
	return params, nil
}

func decodeSessionParams(raw json.RawMessage) (SessionParams, error) {
	fields, err := decodeObject(raw)
	if err != nil {
		return SessionParams{}, err
	}
	if err := requireFields(fields, "session_id"); err != nil {
		return SessionParams{}, err
	}

	var params SessionParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return SessionParams{}, fmt.Errorf("invalid session params: %w", err)
	}
	if params.SessionID == "" {
		return SessionParams{}, errors.New("session_id must be non-empty")
	}
	return params, nil
}

func decodePressParams(raw json.RawMessage) (PressParams, error) {
	fields, err := decodeObject(raw)
	if err != nil {
		return PressParams{}, err
	}
	if err := requireFields(fields, "session_id", "key"); err != nil {
		return PressParams{}, err
	}

	var params PressParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return PressParams{}, fmt.Errorf("invalid press params: %w", err)
	}
	if params.SessionID == "" || params.Key == "" {
		return PressParams{}, errors.New("press session_id and key must be non-empty")
	}
	return params, nil
}

func decodeTypeParams(raw json.RawMessage) (TypeParams, error) {
	fields, err := decodeObject(raw)
	if err != nil {
		return TypeParams{}, err
	}
	if err := requireFields(fields, "session_id", "text"); err != nil {
		return TypeParams{}, err
	}

	var params TypeParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return TypeParams{}, fmt.Errorf("invalid type params: %w", err)
	}
	if params.SessionID == "" {
		return TypeParams{}, errors.New("type session_id must be non-empty")
	}
	return params, nil
}

func decodeWaitParams(raw json.RawMessage) (WaitParams, error) {
	fields, err := decodeObject(raw)
	if err != nil {
		return WaitParams{}, err
	}
	if err := requireFields(fields, "session_id", "timeout_ms"); err != nil {
		return WaitParams{}, err
	}

	var params WaitParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return WaitParams{}, fmt.Errorf("invalid wait params: %w", err)
	}
	if params.SessionID == "" {
		return WaitParams{}, errors.New("wait session_id must be non-empty")
	}
	if params.TimeoutMS < 1 {
		return WaitParams{}, errors.New("wait timeout_ms must be >=1")
	}
	if params.MatchText == "" && params.MatchRE == "" {
		return WaitParams{}, errors.New("wait requires match_text or match_regex")
	}
	return params, nil
}

func decodeSnapshotParams(raw json.RawMessage) (SnapshotParams, error) {
	fields, err := decodeObject(raw)
	if err != nil {
		return SnapshotParams{}, err
	}
	if err := requireFields(fields, "session_id", "name"); err != nil {
		return SnapshotParams{}, err
	}

	var params SnapshotParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return SnapshotParams{}, fmt.Errorf("invalid snapshot params: %w", err)
	}
	if params.SessionID == "" || params.Name == "" {
		return SnapshotParams{}, errors.New("snapshot session_id and name must be non-empty")
	}
	return params, nil
}

func decodeAssertVisualParams(raw json.RawMessage) (AssertVisualParams, error) {
	fields, err := decodeObject(raw)
	if err != nil {
		return AssertVisualParams{}, err
	}
	if err := requireFields(fields, "session_id", "name", "baseline_path"); err != nil {
		return AssertVisualParams{}, err
	}

	var params AssertVisualParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return AssertVisualParams{}, fmt.Errorf("invalid assert-visual params: %w", err)
	}
	if params.SessionID == "" || params.Name == "" || params.BaselinePath == "" {
		return AssertVisualParams{}, errors.New("assert-visual session_id, name, and baseline_path must be non-empty")
	}
	if params.Threshold < 0 || params.Threshold > 1 {
		return AssertVisualParams{}, errors.New("assert-visual threshold must be between 0 and 1")
	}
	return params, nil
}

func decodeRecordParams(raw json.RawMessage) (RecordParams, error) {
	fields, err := decodeObject(raw)
	if err != nil {
		return RecordParams{}, err
	}
	if err := requireFields(fields, "session_id", "output_path"); err != nil {
		return RecordParams{}, err
	}

	var params RecordParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return RecordParams{}, fmt.Errorf("invalid record params: %w", err)
	}
	if params.SessionID == "" || params.OutputPath == "" {
		return RecordParams{}, errors.New("record session_id and output_path must be non-empty")
	}
	return params, nil
}

func decodeObject(raw []byte) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("expected JSON object: %w", err)
	}
	return fields, nil
}

func requireFields(fields map[string]json.RawMessage, names ...string) error {
	for _, name := range names {
		if _, ok := fields[name]; !ok {
			return fmt.Errorf("missing required field: %s", name)
		}
	}
	return nil
}
