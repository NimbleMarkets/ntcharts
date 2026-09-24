package main

import (
	"errors"
	"fmt"
	"image"
	"runtime"
	"sync"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
	_ "github.com/gogpu/wgpu/hal/allbackends"
)

type renderRequest struct {
	preset, width, height                int
	seconds, speed, scale, color, detail float32
}
type uniforms struct {
	Time, Speed, Scale, Color float32
	Width, Height             uint32
	Detail, Pad               float32
}
type frameRenderer interface {
	Render(renderRequest) (*image.NRGBA, error)
}
type gpuJob struct {
	request renderRequest
	reply   chan gpuResult
}
type gpuResult struct {
	image *image.NRGBA
	err   error
}
type gpuRenderer struct {
	mu     sync.Mutex
	jobs   chan gpuJob
	done   chan struct{}
	closed bool
	name   string
}

// All GPU calls, including map/poll and destruction, stay on one OS thread.
// In particular, Metal autorelease pools cannot migrate between Go threads.
func newGPU() (*gpuRenderer, error) {
	g := &gpuRenderer{jobs: make(chan gpuJob), done: make(chan struct{})}
	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(g.done)
		instance, err := wgpu.CreateInstance(nil)
		if err != nil {
			ready <- err
			return
		}
		defer instance.Release()
		adapter, err := instance.RequestAdapter(&wgpu.RequestAdapterOptions{PowerPreference: wgpu.PowerPreferenceHighPerformance})
		if err != nil {
			ready <- err
			return
		}
		if adapter == nil {
			ready <- errors.New("no GPU adapter available")
			return
		}
		defer adapter.Release()
		info := adapter.Info()
		if info.DeviceType == gputypes.DeviceTypeCPU || info.Backend == gputypes.BackendEmpty {
			ready <- errors.New("a hardware GPU is required for this example")
			return
		}
		g.name = info.Name
		dev, err := adapter.RequestDevice(nil)
		if err != nil {
			ready <- err
			return
		}
		defer dev.Release()
		compiled := make([]*gpuPipeline, len(presets))
		defer func() {
			for _, p := range compiled {
				if p != nil {
					p.close()
				}
			}
		}()
		for i, p := range presets {
			compiled[i], err = compileShader(dev, shaderSource(i))
			if err != nil {
				ready <- fmt.Errorf("%s: %w", p.name, err)
				return
			}
		}
		ready <- nil
		for job := range g.jobs {
			r := job.request
			u := uniforms{Time: r.seconds, Speed: r.speed, Scale: r.scale, Color: r.color, Width: uint32(r.width), Height: uint32(r.height), Detail: r.detail}
			p := compiled[r.preset]
			img, err := DispatchRGBA8(dev, ImageDispatch{Label: presets[r.preset].name, Width: r.width, Height: r.height,
				Uniform: BytesOf(&u), BindGroupLayout: p.bindings, Pipeline: p.pipeline})
			job.reply <- gpuResult{img, err}
		}
	}()
	if err := <-ready; err != nil {
		<-g.done
		return nil, err
	}
	return g, nil
}
func (g *gpuRenderer) Render(r renderRequest) (*image.NRGBA, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil, errors.New("GPU renderer closed")
	}
	if r.preset < 0 || r.preset >= len(presets) || r.width < 1 || r.height < 1 || r.width > 2048 || r.height > 1536 {
		return nil, errors.New("invalid render request")
	}
	reply := make(chan gpuResult, 1)
	g.jobs <- gpuJob{r, reply}
	result := <-reply
	return result.image, result.err
}
func (g *gpuRenderer) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.closed {
		g.closed = true
		close(g.jobs)
		<-g.done
	}
}

type gpuPipeline struct {
	shader   *wgpu.ShaderModule
	bindings *wgpu.BindGroupLayout
	layout   *wgpu.PipelineLayout
	pipeline *wgpu.ComputePipeline
}

func (p *gpuPipeline) close() {
	if p.pipeline != nil {
		p.pipeline.Release()
	}
	if p.layout != nil {
		p.layout.Release()
	}
	if p.bindings != nil {
		p.bindings.Release()
	}
	if p.shader != nil {
		p.shader.Release()
	}
}
func compileShader(dev *wgpu.Device, code string) (_ *gpuPipeline, err error) {
	p := &gpuPipeline{}
	defer func() {
		if err != nil {
			p.close()
		}
	}()
	p.shader, err = dev.CreateShaderModule(&wgpu.ShaderModuleDescriptor{WGSL: code})
	if err != nil {
		return nil, err
	}
	p.bindings, err = dev.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{Entries: []wgpu.BindGroupLayoutEntry{
		{Binding: 0, Visibility: wgpu.ShaderStageCompute, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}},
		{Binding: 1, Visibility: wgpu.ShaderStageCompute, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeStorage}},
	}})
	if err != nil {
		return nil, err
	}
	p.layout, err = dev.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{BindGroupLayouts: []*wgpu.BindGroupLayout{p.bindings}})
	if err != nil {
		return nil, err
	}
	p.pipeline, err = dev.CreateComputePipeline(&wgpu.ComputePipelineDescriptor{Layout: p.layout, Module: p.shader, EntryPoint: "main"})
	if err != nil {
		return nil, err
	}
	return p, nil
}
