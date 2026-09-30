package main

import (
	"fmt"
	"math"
	"net/url"
	"runtime"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

// sampleWindow is how many recent frames a summary covers: ten seconds at
// 60 FPS, so startup outliers age out of a steady-state measurement.
const sampleWindow = 600

// samples is a ring of the most recent durations for one stage.
type samples struct {
	values []time.Duration
	next   int
}

func (s *samples) add(d time.Duration) {
	if len(s.values) < sampleWindow {
		s.values = append(s.values, d)
		return
	}
	s.values[s.next] = d
	s.next = (s.next + 1) % sampleWindow
}

// stageSummary describes one stage over the sample window, in milliseconds.
type stageSummary struct {
	Count  int     `json:"count"`
	MeanMS float64 `json:"mean_ms"`
	P50MS  float64 `json:"p50_ms"`
	P95MS  float64 `json:"p95_ms"`
}

func (s *samples) summary() stageSummary {
	n := len(s.values)
	if n == 0 {
		return stageSummary{}
	}
	sorted := slices.Clone(s.values)
	slices.Sort(sorted)
	var total time.Duration
	for _, d := range sorted {
		total += d
	}
	// Nearest-rank percentiles: every reported value is one that was measured.
	rank := func(q float64) float64 { return ms(sorted[int(math.Ceil(q*float64(n)))-1]) }
	return stageSummary{Count: n, MeanMS: ms(total) / float64(n), P50MS: rank(0.50), P95MS: rank(0.95)}
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

// processStart is as early as the program can observe its own start; in the
// browser it follows the download and instantiation of the WebAssembly module.
var processStart = time.Now()

// stageTimes partitions one GPU dispatch: building its buffers, bind group
// and commands; submitting them; waiting for the mapped readback; and copying
// the pixels into the image.
type stageTimes struct{ setup, submit, mapWait, copy time.Duration }

func (s stageTimes) plus(o stageTimes) stageTimes {
	return stageTimes{s.setup + o.setup, s.submit + o.submit, s.mapWait + o.mapWait, s.copy + o.copy}
}

// smooth blends next into s for the header, as the model smooths R and E.
func (s stageTimes) smooth(next stageTimes) stageTimes {
	if s == (stageTimes{}) {
		return next
	}
	blend := func(old, next time.Duration) time.Duration { return (old*4 + next) / 5 }
	return stageTimes{blend(s.setup, next.setup), blend(s.submit, next.submit), blend(s.mapWait, next.mapWait), blend(s.copy, next.copy)}
}

// stageSamples holds the recent samples of every measured stage. render is
// the whole GPU call and contains setup, submit, mapWait and copy; frame is
// the interval between presentations.
type stageSamples struct {
	render, setup, submit, mapWait, copy, encode, view, frame samples
}

func (s *stageSamples) addDispatch(t stageTimes) {
	s.setup.add(t.setup)
	s.submit.add(t.submit)
	s.mapWait.add(t.mapWait)
	s.copy.add(t.copy)
}
func (s *stageSamples) summaries() map[string]stageSummary {
	return map[string]stageSummary{
		"render": s.render.summary(), "setup": s.setup.summary(), "submit": s.submit.summary(),
		"map": s.mapWait.summary(), "copy": s.copy.summary(), "encode": s.encode.summary(),
		"view": s.view.summary(), "frame": s.frame.summary(),
	}
}

// report is the -report document. The browser build logs the same document.
type report struct {
	Compiler      string                  `json:"compiler"`
	GoVersion     string                  `json:"go_version"`
	GPU           string                  `json:"gpu"`
	Transport     string                  `json:"transport"`
	EncodedFrames map[string]int          `json:"encoded_frames"`
	Presets       map[string]int          `json:"rendered_presets"`
	AppFPS        float64                 `json:"app_fps"`
	RenderMS      float64                 `json:"render_ms"`
	EncodeMS      float64                 `json:"encode_ms"`
	Width         int                     `json:"width"`
	Height        int                     `json:"height"`
	Startup       startupReport           `json:"startup"`
	Stages        map[string]stageSummary `json:"stages"`
	Memory        memoryReport            `json:"memory"`
}
type startupReport struct {
	// LoadMS is browser-only: from navigation to the start of the program,
	// which covers downloading, compiling and instantiating the module.
	LoadMS          float64 `json:"load_ms"`
	GPUInitMS       float64 `json:"gpu_init_ms"`
	ShaderCompileMS float64 `json:"shader_compile_ms"`
	FirstFrameMS    float64 `json:"first_frame_ms"`
}
type memoryReport struct {
	HeapSysBytes    uint64 `json:"heap_sys_bytes"`
	HeapAllocBytes  uint64 `json:"heap_alloc_bytes"`
	TotalAllocBytes uint64 `json:"total_alloc_bytes"`
	Mallocs         uint64 `json:"mallocs"`
}

func (m *model) report() report {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return report{
		Compiler: runtime.Compiler, GoVersion: runtime.Version(), GPU: m.gpuName, Transport: m.transport,
		EncodedFrames: m.encodedFrames, Presets: m.renderedPresets,
		AppFPS: m.fps, RenderMS: m.renderMS, EncodeMS: m.encodeMS, Width: m.rasterW, Height: m.rasterH,
		Startup: startupReport{loadMS, ms(m.gpuInit), ms(m.shaderCompile), ms(m.firstFrame)},
		Stages:  m.stages.summaries(),
		Memory:  memoryReport{mem.HeapSys, mem.HeapAlloc, mem.TotalAlloc, mem.Mallocs},
	}
}

// reportInterval is how often the browser build publishes its report.
const reportInterval = 2 * time.Second

type reportMsg struct{}

func reportTick() tea.Cmd {
	return tea.Tick(reportInterval, func(time.Time) tea.Msg { return reportMsg{} })
}

// queryArgs turns a page's query string into flag arguments, so the browser
// build takes the same options as the native one: "?preset=julia&density=24".
// Parameters that are not flags are ignored, and a malformed query yields none.
func queryArgs(query string, known func(name string) bool) []string {
	values, err := url.ParseQuery(strings.TrimPrefix(query, "?"))
	if err != nil {
		return nil
	}
	var args []string
	for name, v := range values {
		if !known(name) {
			continue
		}
		value := v[len(v)-1]
		if value == "" {
			value = "true"
		}
		args = append(args, "-"+name+"="+value)
	}
	slices.Sort(args)
	return args
}

func parseMedium(name string) (picture.KittyMedium, error) {
	switch name {
	case "shm":
		return picture.KittyMediumSharedMemory, nil
	case "direct":
		return picture.KittyMediumDirect, nil
	}
	return 0, fmt.Errorf("-medium=%s: must be shm or direct", name)
}
