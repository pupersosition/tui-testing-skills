package main

import (
	"fmt"

	"github.com/pupersosition/tui-testing-skills/internal/contract"
)

const (
	errorCodeInvalidRequest = "INVALID_REQUEST"
	errorCodeInvalidJSON    = "INVALID_JSON"
	errorCodeInternal       = "INTERNAL_ERROR"
	errorCodeNotImplemented = "NOT_IMPLEMENTED"
)

// SessionRuntime contains dispatcher hooks implemented by the Go PTY runtime workstream.
type SessionRuntime interface {
	Open(params contract.OpenParams) contract.Result
	Close(params contract.SessionParams) contract.Result
	Press(params contract.PressParams) contract.Result
	Type(params contract.TypeParams) contract.Result
	Wait(params contract.WaitParams) contract.Result
}

// VisualRuntime contains dispatcher hooks implemented by the Go visual runtime workstream.
type VisualRuntime interface {
	Snapshot(params contract.SnapshotParams) contract.Result
	AssertVisual(params contract.AssertVisualParams) contract.Result
	Record(params contract.RecordParams) contract.Result
}

// Dispatcher validates request envelopes and routes commands to runtime implementations.
type Dispatcher struct {
	session SessionRuntime
	visual  VisualRuntime
}

func NewDispatcher(session SessionRuntime, visual VisualRuntime) *Dispatcher {
	if session == nil {
		session = NotImplementedSessionRuntime{}
	}
	if visual == nil {
		visual = NotImplementedVisualRuntime{}
	}
	return &Dispatcher{session: session, visual: visual}
}

func (d *Dispatcher) HandleRaw(raw []byte) contract.Result {
	req, err := contract.ParseRequest(raw)
	if err != nil {
		return errorResult("", errorCodeInvalidRequest, err.Error())
	}
	return d.HandleRequest(req)
}

func (d *Dispatcher) HandleRequest(req contract.Request) contract.Result {
	decoded, err := contract.DecodeParams(req.Command, req.Params)
	if err != nil {
		return errorResult("", errorCodeInvalidRequest, err.Error())
	}

	switch req.Command {
	case contract.CommandOpen:
		params := decoded.(contract.OpenParams)
		return normalizeResult(d.session.Open(params), "")
	case contract.CommandClose:
		params := decoded.(contract.SessionParams)
		return normalizeResult(d.session.Close(params), params.SessionID)
	case contract.CommandPress:
		params := decoded.(contract.PressParams)
		return normalizeResult(d.session.Press(params), params.SessionID)
	case contract.CommandType:
		params := decoded.(contract.TypeParams)
		return normalizeResult(d.session.Type(params), params.SessionID)
	case contract.CommandWait:
		params := decoded.(contract.WaitParams)
		return normalizeResult(d.session.Wait(params), params.SessionID)
	case contract.CommandSnapshot:
		params := decoded.(contract.SnapshotParams)
		return normalizeResult(d.visual.Snapshot(params), params.SessionID)
	case contract.CommandAssertVisual:
		params := decoded.(contract.AssertVisualParams)
		return normalizeResult(d.visual.AssertVisual(params), params.SessionID)
	case contract.CommandRecord:
		params := decoded.(contract.RecordParams)
		return normalizeResult(d.visual.Record(params), params.SessionID)
	default:
		return errorResult("", "UNKNOWN_COMMAND", fmt.Sprintf("unsupported command: %q", req.Command))
	}
}

func normalizeResult(result contract.Result, fallbackSessionID string) contract.Result {
	if result.SessionID == "" {
		result.SessionID = fallbackSessionID
	}

	if result.OK {
		result.Error = nil
		return result
	}

	if result.Error == nil {
		result.Error = &contract.ErrorPayload{}
	}
	if result.Error.Code == "" {
		result.Error.Code = errorCodeInternal
	}
	if result.Error.Message == "" {
		result.Error.Message = "command failed"
	}
	return result
}

func errorResult(sessionID string, code string, message string) contract.Result {
	return contract.Result{
		OK:        false,
		SessionID: sessionID,
		Error: &contract.ErrorPayload{
			Code:    code,
			Message: message,
		},
	}
}

// NotImplementedSessionRuntime keeps dispatcher behavior explicit until runtime workstream hooks land.
type NotImplementedSessionRuntime struct{}

func (NotImplementedSessionRuntime) Open(params contract.OpenParams) contract.Result {
	return notImplementedResult("", contract.CommandOpen)
}

func (NotImplementedSessionRuntime) Close(params contract.SessionParams) contract.Result {
	return notImplementedResult(params.SessionID, contract.CommandClose)
}

func (NotImplementedSessionRuntime) Press(params contract.PressParams) contract.Result {
	return notImplementedResult(params.SessionID, contract.CommandPress)
}

func (NotImplementedSessionRuntime) Type(params contract.TypeParams) contract.Result {
	return notImplementedResult(params.SessionID, contract.CommandType)
}

func (NotImplementedSessionRuntime) Wait(params contract.WaitParams) contract.Result {
	return notImplementedResult(params.SessionID, contract.CommandWait)
}

// NotImplementedVisualRuntime keeps dispatcher behavior explicit until visual workstream hooks land.
type NotImplementedVisualRuntime struct{}

func (NotImplementedVisualRuntime) Snapshot(params contract.SnapshotParams) contract.Result {
	return notImplementedResult(params.SessionID, contract.CommandSnapshot)
}

func (NotImplementedVisualRuntime) AssertVisual(params contract.AssertVisualParams) contract.Result {
	return notImplementedResult(params.SessionID, contract.CommandAssertVisual)
}

func (NotImplementedVisualRuntime) Record(params contract.RecordParams) contract.Result {
	return notImplementedResult(params.SessionID, contract.CommandRecord)
}

func notImplementedResult(sessionID string, command contract.Command) contract.Result {
	return errorResult(
		sessionID,
		errorCodeNotImplemented,
		fmt.Sprintf("Go runtime for %q is not wired yet. Use python3 skills/bubbletea-tui-visual-test/scripts/agent_tui.py during migration.", command),
	)
}
