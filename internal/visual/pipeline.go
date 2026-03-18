package visual

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pupersosition/tui-testing-skills/internal/contract"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
var nameSanitizer = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

const (
	defaultRootOutputDir = ".context/artifacts/bubbletea-tui-visual-test"
	rendererUnavailable  = "GIF renderer is unavailable. Ensure a GIF backend is available in this environment."
)

type visualError struct {
	Code    string
	Message string
}

func (e *visualError) Error() string {
	return e.Code + ": " + e.Message
}

func fail(sessionID, code, message string, data map[string]interface{}) contract.Result {
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

func ok(sessionID string, data map[string]interface{}) contract.Result {
	result := contract.Result{
		OK:        true,
		SessionID: sessionID,
	}
	if data != nil {
		result.Data = data
	}
	return result
}

type RuntimeMetadata struct {
	Cols            int    `json:"cols"`
	Rows            int    `json:"rows"`
	Theme           string `json:"theme"`
	ColorMode       string `json:"color_mode"`
	Locale          string `json:"locale"`
	RendererVersion string `json:"renderer_version"`
}

type metadataRecord struct {
	SessionID    string          `json:"session_id"`
	Checkpoint   string          `json:"checkpoint"`
	CreatedAt    string          `json:"created_at"`
	SnapshotPath string          `json:"snapshot_path"`
	Runtime      RuntimeMetadata `json:"runtime"`
}

type diffRecord struct {
	Checkpoint      string  `json:"checkpoint"`
	ActualPath      string  `json:"actual_path"`
	BaselinePath    string  `json:"baseline_path"`
	DifferenceRatio float64 `json:"difference_ratio"`
	Threshold       float64 `json:"threshold"`
	Passed          bool    `json:"passed"`
}

// ErrRendererUnavailable indicates that the environment cannot render GIF output.
var ErrRendererUnavailable = errors.New("renderer unavailable")

type GIFRenderer interface {
	Encode(outputPath string, framePaths []string, frameDurationMS int) error
}

type defaultGIFRenderer struct{}

func (r defaultGIFRenderer) Encode(outputPath string, framePaths []string, frameDurationMS int) error {
	if mode := strings.ToLower(strings.TrimSpace(os.Getenv("BUBBLETEA_GIF_RENDERER"))); mode == "disabled" || mode == "unavailable" {
		return ErrRendererUnavailable
	}

	delay := int(math.Round(float64(frameDurationMS) / 10.0))
	if delay < 1 {
		delay = 1
	}

	frames := make([]*image.Paletted, 0, len(framePaths))
	delays := make([]int, 0, len(framePaths))
	for _, framePath := range framePaths {
		src, err := readPNG(framePath)
		if err != nil {
			return err
		}

		dst := image.NewPaletted(src.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(dst, src.Bounds(), src, src.Bounds().Min)
		frames = append(frames, dst)
		delays = append(delays, delay)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return gif.EncodeAll(file, &gif.GIF{
		Image:     frames,
		Delay:     delays,
		LoopCount: 0,
	})
}

type Config struct {
	RunDir        string
	RootOutputDir string
	Now           func() time.Time
	GIFRenderer   GIFRenderer
}

type Pipeline struct {
	runDir       string
	snapshotsDir string
	metadataDir  string
	diffsDir     string
	gifsDir      string

	checkpointIndex map[string]string
	checkpointOrder []string

	now      func() time.Time
	renderer GIFRenderer
}

func New(config Config) (*Pipeline, error) {
	now := config.Now
	if now == nil {
		now = time.Now
	}

	renderer := config.GIFRenderer
	if renderer == nil {
		renderer = defaultGIFRenderer{}
	}

	runDir := strings.TrimSpace(config.RunDir)
	if runDir == "" {
		root := strings.TrimSpace(config.RootOutputDir)
		if root == "" {
			root = defaultRootOutputDir
		}
		runID, err := makeRunID(now())
		if err != nil {
			return nil, err
		}
		runDir = filepath.Join(root, runID)
	}

	absRunDir, err := filepath.Abs(runDir)
	if err != nil {
		return nil, err
	}

	p := &Pipeline{
		runDir:          absRunDir,
		snapshotsDir:    filepath.Join(absRunDir, "snapshots"),
		metadataDir:     filepath.Join(absRunDir, "metadata"),
		diffsDir:        filepath.Join(absRunDir, "diffs"),
		gifsDir:         filepath.Join(absRunDir, "gifs"),
		checkpointIndex: map[string]string{},
		checkpointOrder: []string{},
		now:             now,
		renderer:        renderer,
	}
	for _, dir := range []string{
		filepath.Join(absRunDir, "logs"),
		p.snapshotsDir,
		p.metadataDir,
		p.diffsDir,
		p.gifsDir,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (p *Pipeline) RunDir() string {
	return p.runDir
}

func (p *Pipeline) Snapshot(sessionID, name, screenText string, runtime map[string]interface{}) contract.Result {
	cleanName, err := sanitizeCheckpointName(name)
	if err != nil {
		return visualResultError(sessionID, err)
	}
	metadata, err := validateRuntimeMetadata(runtime)
	if err != nil {
		return visualResultError(sessionID, err)
	}

	lines := normalizedScreenLines(screenText, metadata.Cols, metadata.Rows)
	img := renderScreen(lines, metadata.Cols, metadata.Rows)

	snapshotPath := filepath.Join(p.snapshotsDir, cleanName+".png")
	if err := writePNG(snapshotPath, img); err != nil {
		return fail(sessionID, "render_error", err.Error(), nil)
	}

	absSnapshotPath, err := filepath.Abs(snapshotPath)
	if err != nil {
		return fail(sessionID, "render_error", err.Error(), nil)
	}
	metadataPath := filepath.Join(p.metadataDir, cleanName+".json")
	absMetadataPath, err := filepath.Abs(metadataPath)
	if err != nil {
		return fail(sessionID, "render_error", err.Error(), nil)
	}

	record := metadataRecord{
		SessionID:    sessionID,
		Checkpoint:   cleanName,
		CreatedAt:    timestampUTC(p.now()),
		SnapshotPath: absSnapshotPath,
		Runtime:      metadata,
	}
	if err := writeJSON(metadataPath, record); err != nil {
		return fail(sessionID, "render_error", err.Error(), nil)
	}

	if _, exists := p.checkpointIndex[cleanName]; !exists {
		p.checkpointOrder = append(p.checkpointOrder, cleanName)
	}
	p.checkpointIndex[cleanName] = absSnapshotPath

	return ok(sessionID, map[string]interface{}{
		"run_dir":       p.runDir,
		"snapshot_path": absSnapshotPath,
		"metadata_path": absMetadataPath,
	})
}

func (p *Pipeline) AssertVisual(sessionID, name, baselinePath string, threshold float64) contract.Result {
	if threshold < 0 || threshold > 1 {
		return fail(sessionID, "invalid_threshold", "Threshold must be in the [0, 1] range.", nil)
	}
	cleanName, err := sanitizeCheckpointName(name)
	if err != nil {
		return visualResultError(sessionID, err)
	}

	actualPath := p.resolveActualPath(cleanName)
	if _, err := os.Stat(actualPath); err != nil {
		return fail(sessionID, "missing_snapshot", fmt.Sprintf("Snapshot for checkpoint '%s' was not found.", cleanName), nil)
	}

	absActualPath, err := filepath.Abs(actualPath)
	if err != nil {
		return fail(sessionID, "assert_failed", err.Error(), nil)
	}
	absBaselinePath, err := filepath.Abs(baselinePath)
	if err != nil {
		return fail(sessionID, "assert_failed", err.Error(), nil)
	}
	if _, err := os.Stat(absBaselinePath); err != nil {
		return fail(sessionID, "missing_baseline", fmt.Sprintf("Baseline PNG does not exist: %s", absBaselinePath), nil)
	}

	actualImage, err := readPNG(absActualPath)
	if err != nil {
		return fail(sessionID, "invalid_png", err.Error(), nil)
	}
	baselineImage, err := readPNG(absBaselinePath)
	if err != nil {
		return fail(sessionID, "invalid_png", err.Error(), nil)
	}

	differenceRatio := 0.0
	passed := true
	var diffArtifact interface{}

	if !actualImage.Rect.Eq(baselineImage.Rect) {
		differenceRatio = 1.0
		passed = false
		diffPath := filepath.Join(p.diffsDir, cleanName+".size-mismatch.json")
		diffPayload := map[string]interface{}{
			"checkpoint":    cleanName,
			"actual_size":   []int{actualImage.Rect.Dx(), actualImage.Rect.Dy()},
			"baseline_size": []int{baselineImage.Rect.Dx(), baselineImage.Rect.Dy()},
		}
		if err := writeJSON(diffPath, diffPayload); err != nil {
			return fail(sessionID, "assert_failed", err.Error(), nil)
		}
		absDiffPath, err := filepath.Abs(diffPath)
		if err != nil {
			return fail(sessionID, "assert_failed", err.Error(), nil)
		}
		diffArtifact = absDiffPath
	} else {
		diffImage, ratio := pixelDiff(actualImage, baselineImage)
		differenceRatio = ratio
		passed = ratio <= threshold

		if !passed {
			diffPath := filepath.Join(p.diffsDir, cleanName+".diff.png")
			if err := writePNG(diffPath, diffImage); err != nil {
				return fail(sessionID, "assert_failed", err.Error(), nil)
			}

			metadataPath := filepath.Join(p.diffsDir, cleanName+".diff.json")
			if err := writeJSON(metadataPath, diffRecord{
				Checkpoint:      cleanName,
				ActualPath:      absActualPath,
				BaselinePath:    absBaselinePath,
				DifferenceRatio: differenceRatio,
				Threshold:       threshold,
				Passed:          passed,
			}); err != nil {
				return fail(sessionID, "assert_failed", err.Error(), nil)
			}

			absDiffPath, err := filepath.Abs(diffPath)
			if err != nil {
				return fail(sessionID, "assert_failed", err.Error(), nil)
			}
			diffArtifact = absDiffPath
		}
	}

	return ok(sessionID, map[string]interface{}{
		"checkpoint":       cleanName,
		"actual_path":      absActualPath,
		"baseline_path":    absBaselinePath,
		"difference_ratio": differenceRatio,
		"threshold":        threshold,
		"passed":           passed,
		"diff_artifact":    diffArtifact,
	})
}

func (p *Pipeline) Record(sessionID, outputPath string, framePaths []string, frameDurationMS int) contract.Result {
	if frameDurationMS <= 0 {
		return fail(sessionID, "invalid_frame_duration", "frame_duration_ms must be a positive integer.", nil)
	}

	frames := framePaths
	if len(frames) == 0 {
		frames = p.defaultFrames()
	}
	if len(frames) == 0 {
		return fail(sessionID, "no_frames", "No frames available. Capture snapshots before calling record.", nil)
	}

	normalizedFrames := make([]string, 0, len(frames))
	for _, frame := range frames {
		absFrame, err := filepath.Abs(frame)
		if err != nil {
			return fail(sessionID, "missing_frame", err.Error(), nil)
		}
		if _, err := os.Stat(absFrame); err != nil {
			return fail(sessionID, "missing_frame", fmt.Sprintf("Frame PNG does not exist: %s", absFrame), nil)
		}
		normalizedFrames = append(normalizedFrames, absFrame)
	}

	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fail(sessionID, "record_failed", err.Error(), nil)
	}
	if err := os.MkdirAll(filepath.Dir(absOutputPath), 0o755); err != nil {
		return fail(sessionID, "record_failed", err.Error(), nil)
	}

	if err := p.renderer.Encode(absOutputPath, normalizedFrames, frameDurationMS); err != nil {
		if errors.Is(err, ErrRendererUnavailable) {
			return fail(sessionID, "renderer_unavailable", rendererUnavailable, nil)
		}
		return fail(sessionID, "record_failed", err.Error(), nil)
	}

	return ok(sessionID, map[string]interface{}{
		"output_path":       absOutputPath,
		"frame_count":       len(normalizedFrames),
		"frame_duration_ms": frameDurationMS,
	})
}

func (p *Pipeline) resolveActualPath(checkpointName string) string {
	if path, exists := p.checkpointIndex[checkpointName]; exists {
		return path
	}
	return filepath.Join(p.snapshotsDir, checkpointName+".png")
}

func (p *Pipeline) defaultFrames() []string {
	frames := make([]string, 0, len(p.checkpointOrder))
	for _, checkpoint := range p.checkpointOrder {
		path := p.checkpointIndex[checkpoint]
		if path != "" {
			frames = append(frames, path)
		}
	}
	return frames
}

func visualResultError(sessionID string, err error) contract.Result {
	var vErr *visualError
	if errors.As(err, &vErr) {
		return fail(sessionID, vErr.Code, vErr.Message, nil)
	}
	return fail(sessionID, "visual_error", err.Error(), nil)
}

func sanitizeCheckpointName(name string) (string, error) {
	cleaned := strings.Trim(nameSanitizer.ReplaceAllString(name, "-"), "-")
	if cleaned == "" {
		return "", &visualError{
			Code:    "invalid_name",
			Message: "Checkpoint name must contain letters, numbers, dots, dashes, or underscores.",
		}
	}
	return cleaned, nil
}

func validateRuntimeMetadata(metadata map[string]interface{}) (RuntimeMetadata, error) {
	if metadata == nil {
		return RuntimeMetadata{}, &visualError{
			Code:    "invalid_runtime_metadata",
			Message: "Missing required runtime metadata fields: cols, rows, theme, color_mode, locale, renderer_version.",
		}
	}

	cols, ok := asPositiveInt(metadata["cols"])
	if !ok {
		return RuntimeMetadata{}, &visualError{
			Code:    "invalid_runtime_metadata",
			Message: "Runtime metadata field 'cols' must be a positive integer.",
		}
	}
	rows, ok := asPositiveInt(metadata["rows"])
	if !ok {
		return RuntimeMetadata{}, &visualError{
			Code:    "invalid_runtime_metadata",
			Message: "Runtime metadata field 'rows' must be a positive integer.",
		}
	}

	theme := strings.TrimSpace(asString(metadata["theme"]))
	colorMode := strings.TrimSpace(asString(metadata["color_mode"]))
	locale := strings.TrimSpace(asString(metadata["locale"]))
	rendererVersion := strings.TrimSpace(asString(metadata["renderer_version"]))
	missing := []string{}
	if theme == "" {
		missing = append(missing, "theme")
	}
	if colorMode == "" {
		missing = append(missing, "color_mode")
	}
	if locale == "" {
		missing = append(missing, "locale")
	}
	if rendererVersion == "" {
		missing = append(missing, "renderer_version")
	}
	if len(missing) > 0 {
		return RuntimeMetadata{}, &visualError{
			Code:    "invalid_runtime_metadata",
			Message: "Missing required runtime metadata fields: " + strings.Join(missing, ", ") + ".",
		}
	}

	return RuntimeMetadata{
		Cols:            cols,
		Rows:            rows,
		Theme:           theme,
		ColorMode:       colorMode,
		Locale:          locale,
		RendererVersion: rendererVersion,
	}, nil
}

func asPositiveInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, v > 0
	case int64:
		return int(v), v > 0
	case float64:
		i := int(v)
		return i, v == float64(i) && i > 0
	default:
		return 0, false
	}
}

func asString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func normalizedScreenLines(screenText string, cols, rows int) []string {
	clean := ansiEscape.ReplaceAllString(screenText, "")
	clean = strings.ReplaceAll(clean, "\r", "\n")
	logicalLines := strings.Split(clean, "\n")

	start := 0
	if len(logicalLines) > rows {
		start = len(logicalLines) - rows
	}
	window := logicalLines[start:]

	lines := make([]string, 0, rows)
	for idx := 0; idx < rows; idx++ {
		line := ""
		if idx < len(window) {
			line = window[idx]
		}
		line = expandTabs(line, 4)
		runes := []rune(line)
		if len(runes) > cols {
			runes = runes[:cols]
		}
		if len(runes) < cols {
			padding := make([]rune, cols-len(runes))
			for i := range padding {
				padding[i] = ' '
			}
			runes = append(runes, padding...)
		}
		lines = append(lines, string(runes))
	}
	return lines
}

func expandTabs(line string, width int) string {
	var b strings.Builder
	col := 0
	for _, r := range line {
		if r == '\t' {
			spaces := width - (col % width)
			for i := 0; i < spaces; i++ {
				b.WriteByte(' ')
			}
			col += spaces
			continue
		}
		b.WriteRune(r)
		col++
	}
	return b.String()
}

func renderScreen(lines []string, cols, rows int) *image.NRGBA {
	cellWidth := 8
	cellHeight := 14
	width := max(1, cols*cellWidth)
	height := max(1, rows*cellHeight)
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	bg := color.NRGBA{R: 24, G: 24, B: 24, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	for rowIndex, line := range lines {
		runes := []rune(line)
		for colIndex, ch := range runes {
			var px color.NRGBA
			if ch >= 32 && ch <= 126 && ch != ' ' {
				// Include rune identity in the grayscale value so numeric/text changes are detectable.
				v := uint8(120 + (int(ch) % 110))
				px = color.NRGBA{R: v, G: v, B: v, A: 255}
			} else {
				px = color.NRGBA{R: 30, G: 30, B: 30, A: 255}
			}

			left := colIndex * cellWidth
			top := rowIndex * cellHeight
			rect := image.Rect(left, top, left+cellWidth, top+cellHeight)
			draw.Draw(img, rect, &image.Uniform{C: px}, image.Point{}, draw.Src)
		}
	}
	return img
}

func readPNG(path string) (*image.NRGBA, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}

	nrgba := image.NewNRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), img, img.Bounds().Min, draw.Src)
	return nrgba, nil
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

func writeJSON(path string, value interface{}) error {
	blob, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	blob = append(blob, '\n')
	return os.WriteFile(path, blob, 0o644)
}

func pixelDiff(actual, baseline *image.NRGBA) (*image.NRGBA, float64) {
	bounds := actual.Bounds()
	diff := image.NewNRGBA(bounds)
	mismatched := 0
	total := bounds.Dx() * bounds.Dy()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			a := actual.NRGBAAt(x, y)
			b := baseline.NRGBAAt(x, y)
			if a != b {
				mismatched++
				diff.SetNRGBA(x, y, color.NRGBA{R: 255, G: 32, B: 32, A: 255})
				continue
			}

			neutral := uint8((uint16(a.R) + uint16(a.G) + uint16(a.B)) / 3)
			diff.SetNRGBA(x, y, color.NRGBA{R: neutral, G: neutral, B: neutral, A: 255})
		}
	}
	if total == 0 {
		return diff, 0
	}
	return diff, float64(mismatched) / float64(total)
}

func timestampUTC(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

func makeRunID(now time.Time) (string, error) {
	stamp := now.UTC().Format("20060102T150405Z")
	entropy := make([]byte, 3)
	if _, err := rand.Read(entropy); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d-%s", stamp, os.Getpid(), hex.EncodeToString(entropy)), nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
