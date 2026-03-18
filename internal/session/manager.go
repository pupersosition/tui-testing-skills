package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"

	"github.com/pupersosition/tui-testing-skills/internal/contract"
)

const (
	defaultTerm        = "xterm-256color"
	defaultRenderer    = "builtin-terminal-rasterizer/1.0"
	maxTranscriptBytes = 1 << 20
)

// Option customizes Manager behavior.
type Option func(*Manager)

// Manager owns PTY-backed command sessions for open/close/press/type/wait.
type Manager struct {
	mu         sync.RWMutex
	sessions   map[string]*sessionState
	idFactory  func() string
	now        func() time.Time
	sleep      func(time.Duration)
	waitPoll   time.Duration
	termSignal syscall.Signal
}

type sessionState struct {
	id   string
	cmd  *exec.Cmd
	ptmx *os.File

	command string
	cwd     string
	cols    int
	rows    int
	env     map[string]string

	outputMu   sync.RWMutex
	transcript []byte
	lastScreen string

	exited chan struct{}
}

// NewManager creates a session manager with deterministic defaults.
func NewManager(opts ...Option) *Manager {
	mgr := &Manager{
		sessions:   make(map[string]*sessionState),
		idFactory:  randomID,
		now:        time.Now,
		sleep:      time.Sleep,
		waitPoll:   20 * time.Millisecond,
		termSignal: syscall.SIGTERM,
	}
	for _, opt := range opts {
		opt(mgr)
	}
	return mgr
}

// WithIDFactory overrides session ID generation.
func WithIDFactory(fn func() string) Option {
	return func(m *Manager) {
		if fn != nil {
			m.idFactory = fn
		}
	}
}

// WithClock overrides clock/sleep behavior for tests.
func WithClock(now func() time.Time, sleep func(time.Duration)) Option {
	return func(m *Manager) {
		if now != nil {
			m.now = now
		}
		if sleep != nil {
			m.sleep = sleep
		}
	}
}

// ExecuteRequest runs an already-validated contract request.
func (m *Manager) ExecuteRequest(req contract.Request) contract.Result {
	return m.Execute(req.Command, req.Params)
}

// Execute decodes command params and dispatches to the session runtime handlers.
func (m *Manager) Execute(command contract.Command, raw json.RawMessage) contract.Result {
	decoded, err := contract.DecodeParams(command, raw)
	if err != nil {
		return failure("", "INVALID_PARAMS", err.Error(), nil)
	}

	switch command {
	case contract.CommandOpen:
		return m.Open(decoded.(contract.OpenParams))
	case contract.CommandClose:
		return m.Close(decoded.(contract.SessionParams))
	case contract.CommandPress:
		return m.Press(decoded.(contract.PressParams))
	case contract.CommandType:
		return m.Type(decoded.(contract.TypeParams))
	case contract.CommandWait:
		return m.Wait(decoded.(contract.WaitParams))
	default:
		return failure("", "UNKNOWN_COMMAND", fmt.Sprintf("Unsupported command: %s", command), nil)
	}
}

// Open starts a new PTY-backed process session.
func (m *Manager) Open(params contract.OpenParams) contract.Result {
	envMap := normalizeEnv(params)
	cmd := exec.Command("/bin/sh", "-lc", params.Cmd)
	cmd.Dir = params.Cwd
	cmd.Env = envMapToSlice(envMap)

	ws := &pty.Winsize{
		Rows: uint16(params.Rows),
		Cols: uint16(params.Cols),
	}
	ptmx, err := pty.StartWithSize(cmd, ws)
	if err != nil {
		return failure("", "OPEN_FAILED", err.Error(), nil)
	}

	sid := m.nextSessionID()
	state := &sessionState{
		id:      sid,
		cmd:     cmd,
		ptmx:    ptmx,
		command: params.Cmd,
		cwd:     params.Cwd,
		cols:    params.Cols,
		rows:    params.Rows,
		env:     copyMap(envMap),
		exited:  make(chan struct{}),
	}

	m.mu.Lock()
	m.sessions[sid] = state
	m.mu.Unlock()

	go state.captureOutput()
	go state.awaitExit()

	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}

	return success(sid, map[string]interface{}{
		"pid":  pid,
		"cmd":  params.Cmd,
		"cwd":  params.Cwd,
		"cols": params.Cols,
		"rows": params.Rows,
	})
}

// Close terminates an active session.
func (m *Manager) Close(params contract.SessionParams) contract.Result {
	state, ok := m.takeSession(params.SessionID)
	if !ok {
		return failure(params.SessionID, "SESSION_NOT_FOUND", "Unknown session: "+params.SessionID, nil)
	}

	terminated := false
	var closeErr error

	if state.cmd.Process != nil && isRunning(state.cmd) {
		if err := state.cmd.Process.Signal(m.termSignal); err == nil || errors.Is(err, os.ErrProcessDone) {
			terminated = true
		}
	}

	if err := state.ptmx.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		closeErr = err
	}

	if !waitFor(state.exited, 1250*time.Millisecond) {
		if state.cmd.Process != nil {
			if err := state.cmd.Process.Kill(); err == nil || errors.Is(err, os.ErrProcessDone) {
				terminated = true
			}
		}
		if !waitFor(state.exited, 1250*time.Millisecond) {
			if closeErr == nil {
				closeErr = errors.New("session process did not exit after close")
			}
		}
	}

	exitStatus, signalStatus := processExitDetails(state.cmd.ProcessState)
	if closeErr != nil {
		return failure(state.id, "CLOSE_FAILED", closeErr.Error(), nil)
	}

	data := map[string]interface{}{
		"closed":       true,
		"terminated":   terminated,
		"exitstatus":   exitStatus,
		"signalstatus": signalStatus,
	}
	return success(state.id, data)
}

// Press sends a mapped key sequence to an active session.
func (m *Manager) Press(params contract.PressParams) contract.Result {
	state, ok := m.getSession(params.SessionID)
	if !ok {
		return failure(params.SessionID, "SESSION_NOT_FOUND", "Unknown session: "+params.SessionID, nil)
	}

	keyBytes, err := mapKey(params.Key)
	if err != nil {
		return failure(params.SessionID, "INVALID_PARAMS", err.Error(), nil)
	}
	if _, err := state.ptmx.Write(keyBytes); err != nil {
		return failure(params.SessionID, "INTERACTION_FAILED", err.Error(), nil)
	}
	return success(params.SessionID, map[string]interface{}{
		"action": "press",
		"key":    params.Key,
	})
}

// Type writes text to an active session.
func (m *Manager) Type(params contract.TypeParams) contract.Result {
	state, ok := m.getSession(params.SessionID)
	if !ok {
		return failure(params.SessionID, "SESSION_NOT_FOUND", "Unknown session: "+params.SessionID, nil)
	}
	if _, err := io.WriteString(state.ptmx, params.Text); err != nil {
		return failure(params.SessionID, "INTERACTION_FAILED", err.Error(), nil)
	}
	return success(params.SessionID, map[string]interface{}{
		"action": "type",
		"text":   params.Text,
		"bytes":  len(params.Text),
	})
}

// Wait blocks until text/regex appears in transcript output or times out.
func (m *Manager) Wait(params contract.WaitParams) contract.Result {
	state, ok := m.getSession(params.SessionID)
	if !ok {
		return failure(params.SessionID, "SESSION_NOT_FOUND", "Unknown session: "+params.SessionID, nil)
	}

	if params.TimeoutMS <= 0 {
		return failure(params.SessionID, "INVALID_PARAMS", "wait requires timeout_ms > 0", nil)
	}

	matchText := params.MatchText
	matchRE := params.MatchRE
	if matchText == "" && matchRE == "" {
		return failure(params.SessionID, "INVALID_PARAMS", "wait requires match_text or match_regex", nil)
	}

	mode := "text"
	var re *regexp.Regexp
	if matchText == "" {
		mode = "regex"
		compiled, err := regexp.Compile(matchRE)
		if err != nil {
			return failure(params.SessionID, "INVALID_REGEX", err.Error(), nil)
		}
		re = compiled
	}

	start := m.now()
	timeout := time.Duration(params.TimeoutMS) * time.Millisecond
	deadline := start.Add(timeout)

	for {
		screen := state.transcriptSnapshot()
		matched, matchedValue := waitMatched(screen, matchText, re)
		if matched {
			if trimmed := normalizeScreen(screen, state.cols, state.rows); trimmed != "" {
				state.setLastScreen(trimmed)
			}
			return success(params.SessionID, map[string]interface{}{
				"mode":          mode,
				"matched":       true,
				"matched_value": matchedValue,
				"elapsed_ms":    int(m.now().Sub(start).Milliseconds()),
			})
		}

		select {
		case <-state.exited:
			screen = state.transcriptSnapshot()
			matched, matchedValue = waitMatched(screen, matchText, re)
			if matched {
				return success(params.SessionID, map[string]interface{}{
					"mode":          mode,
					"matched":       true,
					"matched_value": matchedValue,
					"elapsed_ms":    int(m.now().Sub(start).Milliseconds()),
				})
			}
			return failure(params.SessionID, "SESSION_ENDED", "session ended before wait condition matched", nil)
		default:
		}

		if !m.now().Before(deadline) {
			return failure(params.SessionID, "WAIT_TIMEOUT", fmt.Sprintf("wait timed out after %dms", params.TimeoutMS), map[string]interface{}{
				"timeout_ms": params.TimeoutMS,
				"mode":       mode,
			})
		}
		m.sleep(m.waitPoll)
	}
}

// RuntimeMetadata returns deterministic runtime metadata used by visual tooling.
func (m *Manager) RuntimeMetadata(sessionID string) (map[string]interface{}, bool) {
	state, ok := m.getSession(sessionID)
	if !ok {
		return nil, false
	}

	locale := state.env["LC_ALL"]
	if locale == "" {
		locale = state.env["LANG"]
	}

	theme := state.env["BUBBLETEA_THEME"]
	if theme == "" {
		theme = "default"
	}

	colorMode := state.env["COLORTERM"]
	if colorMode == "" {
		colorMode = "256"
	}

	return map[string]interface{}{
		"cols":             state.cols,
		"rows":             state.rows,
		"locale":           locale,
		"theme":            theme,
		"color_mode":       colorMode,
		"renderer_version": defaultRenderer,
	}, true
}

// ScreenText returns the latest normalized transcript text.
func (m *Manager) ScreenText(sessionID string) (string, bool) {
	state, ok := m.getSession(sessionID)
	if !ok {
		return "", false
	}

	current := normalizeScreen(state.transcriptSnapshot(), state.cols, state.rows)
	if current != "" {
		state.setLastScreen(current)
		return current, true
	}
	return state.lastKnownScreen(), true
}

// HasSession reports whether a session is active.
func (m *Manager) HasSession(sessionID string) bool {
	_, ok := m.getSession(sessionID)
	return ok
}

// ActiveSessionIDs returns sorted active session IDs.
func (m *Manager) ActiveSessionIDs() []string {
	m.mu.RLock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.RUnlock()
	sort.Strings(ids)
	return ids
}

// Shutdown closes all active sessions.
func (m *Manager) Shutdown() {
	for _, id := range m.ActiveSessionIDs() {
		_ = m.Close(contract.SessionParams{SessionID: id})
	}
}

func (m *Manager) getSession(sessionID string) (*sessionState, bool) {
	m.mu.RLock()
	state, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	return state, ok
}

func (m *Manager) takeSession(sessionID string) (*sessionState, bool) {
	m.mu.Lock()
	state, ok := m.sessions[sessionID]
	if ok {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()
	return state, ok
}

func (m *Manager) nextSessionID() string {
	for {
		candidate := "session-" + m.idFactory()
		m.mu.RLock()
		_, exists := m.sessions[candidate]
		m.mu.RUnlock()
		if !exists {
			return candidate
		}
	}
}

func (s *sessionState) captureOutput() {
	buf := make([]byte, 4096)
	for {
		n, err := s.ptmx.Read(buf)
		if n > 0 {
			s.appendTranscript(buf[:n])
		}
		if err != nil {
			if errors.Is(err, os.ErrClosed) || errors.Is(err, io.EOF) {
				return
			}
			return
		}
	}
}

func (s *sessionState) awaitExit() {
	_ = s.cmd.Wait()
	close(s.exited)
}

func (s *sessionState) appendTranscript(chunk []byte) {
	if len(chunk) == 0 {
		return
	}
	s.outputMu.Lock()
	s.transcript = append(s.transcript, chunk...)
	if len(s.transcript) > maxTranscriptBytes {
		s.transcript = append([]byte(nil), s.transcript[len(s.transcript)-maxTranscriptBytes:]...)
	}
	s.outputMu.Unlock()
}

func (s *sessionState) transcriptSnapshot() string {
	s.outputMu.RLock()
	out := string(s.transcript)
	s.outputMu.RUnlock()
	return out
}

func (s *sessionState) setLastScreen(text string) {
	s.outputMu.Lock()
	s.lastScreen = text
	s.outputMu.Unlock()
}

func (s *sessionState) lastKnownScreen() string {
	s.outputMu.RLock()
	text := s.lastScreen
	s.outputMu.RUnlock()
	return text
}

func waitMatched(transcript string, matchText string, re *regexp.Regexp) (bool, string) {
	if matchText != "" {
		if strings.Contains(transcript, matchText) {
			return true, matchText
		}
		return false, ""
	}
	if re == nil {
		return false, ""
	}
	matched := re.FindString(transcript)
	if matched == "" {
		return false, ""
	}
	return true, matched
}

func normalizeEnv(params contract.OpenParams) map[string]string {
	env := make(map[string]string)
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		env[parts[0]] = parts[1]
	}
	for k, v := range params.Env {
		env[k] = v
	}

	if params.Locale != "" {
		env["LANG"] = params.Locale
		env["LC_ALL"] = params.Locale
	}
	if params.ColorMode == "truecolor" || params.ColorMode == "16" || params.ColorMode == "256" {
		env["COLORTERM"] = params.ColorMode
	}
	if params.Theme != "" {
		env["BUBBLETEA_THEME"] = params.Theme
	}
	if env["TERM"] == "" {
		env["TERM"] = defaultTerm
	}
	return env
}

func normalizeScreen(transcript string, cols int, rows int) string {
	text := strings.TrimSpace(transcript)
	if text == "" {
		return ""
	}
	maxChars := cols * rows * 4
	if maxChars < 2048 {
		maxChars = 2048
	}
	if len(text) > maxChars {
		text = text[len(text)-maxChars:]
	}
	return text
}

func envMapToSlice(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+values[key])
	}
	return out
}

func mapKey(key string) ([]byte, error) {
	if key == "" {
		return nil, errors.New("press requires non-empty key")
	}

	normalized := strings.ToLower(strings.TrimSpace(key))
	switch normalized {
	case "enter":
		return []byte("\r"), nil
	case "tab":
		return []byte("\t"), nil
	case "esc":
		return []byte("\x1b"), nil
	case "backspace":
		return []byte("\x7f"), nil
	case "up":
		return []byte("\x1b[A"), nil
	case "down":
		return []byte("\x1b[B"), nil
	case "right":
		return []byte("\x1b[C"), nil
	case "left":
		return []byte("\x1b[D"), nil
	}

	if strings.HasPrefix(normalized, "ctrl+") && len(normalized) == len("ctrl+x") {
		ch := normalized[len(normalized)-1]
		if ch >= 'a' && ch <= 'z' {
			return []byte{ch - 'a' + 1}, nil
		}
	}

	return []byte(key), nil
}

func success(sessionID string, data map[string]interface{}) contract.Result {
	result := contract.Result{
		OK:        true,
		SessionID: sessionID,
	}
	if data != nil {
		result.Data = data
	}
	return result
}

func failure(sessionID string, code string, message string, data map[string]interface{}) contract.Result {
	result := contract.Result{
		OK:        false,
		SessionID: sessionID,
		Error: &contract.ErrorPayload{
			Code:    code,
			Message: message,
		},
	}
	if data != nil {
		result.Data = data
	}
	return result
}

func randomID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func copyMap(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func processExitDetails(state *os.ProcessState) (interface{}, interface{}) {
	if state == nil {
		return nil, nil
	}

	waitStatus, ok := state.Sys().(syscall.WaitStatus)
	if !ok {
		exitCode := state.ExitCode()
		if exitCode >= 0 {
			return exitCode, nil
		}
		return nil, nil
	}

	if waitStatus.Exited() {
		return waitStatus.ExitStatus(), nil
	}
	if waitStatus.Signaled() {
		return nil, waitStatus.Signal().String()
	}
	return nil, nil
}

func waitFor(ch <-chan struct{}, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ch:
		return true
	case <-timer.C:
		return false
	}
}

func isRunning(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return false
	}
	err := cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}
