package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"unsafe"

	"github.com/gogpu/wgpu"
)

// BytesOf returns the byte representation of v for uniform uploads. The caller
// must use the returned slice before v goes out of scope.
func BytesOf[T any](v *T) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(v)), int(unsafe.Sizeof(*v)))
}

// ImageDispatch describes a compute shader that writes one packed rgba8 u32 per
// output pixel to a storage buffer.
type ImageDispatch struct {
	Label string

	Width, Height int
	WorkgroupX    uint32
	WorkgroupY    uint32

	Uniform []byte

	BindGroupLayout *wgpu.BindGroupLayout
	Pipeline        *wgpu.ComputePipeline

	UniformBinding uint32
	OutputBinding  uint32
}

// DispatchRGBA8 runs req.Pipeline over Width x Height and reads back a packed
// RGBA8 storage buffer into image.NRGBA. It creates only per-dispatch resources.
func DispatchRGBA8(dev *wgpu.Device, req ImageDispatch) (*image.NRGBA, error) {
	if dev == nil {
		return nil, errors.New("nil device")
	}
	if req.Pipeline == nil {
		return nil, errors.New("nil compute pipeline")
	}
	if req.BindGroupLayout == nil {
		return nil, errors.New("nil bind group layout")
	}
	if len(req.Uniform) == 0 {
		return nil, errors.New("empty uniform")
	}
	if req.Width <= 0 || req.Height <= 0 {
		return image.NewNRGBA(image.Rect(0, 0, max(req.Width, 0), max(req.Height, 0))), nil
	}
	if req.WorkgroupX == 0 {
		req.WorkgroupX = 8
	}
	if req.WorkgroupY == 0 {
		req.WorkgroupY = 8
	}
	if req.UniformBinding == 0 && req.OutputBinding == 0 {
		req.OutputBinding = 1
	}

	q := dev.Queue()
	npix := req.Width * req.Height
	outBytes := uint64(npix * 4)

	uni, err := dev.CreateBuffer(&wgpu.BufferDescriptor{
		Label: req.Label + ":uniform",
		Size:  uint64(len(req.Uniform)),
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return nil, fmt.Errorf("create uniform buffer: %w", err)
	}
	defer uni.Release()

	out, err := dev.CreateBuffer(&wgpu.BufferDescriptor{
		Label: req.Label + ":out",
		Size:  outBytes,
		Usage: wgpu.BufferUsageStorage | wgpu.BufferUsageCopySrc | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return nil, fmt.Errorf("create output buffer: %w", err)
	}
	defer out.Release()

	staging, err := dev.CreateBuffer(&wgpu.BufferDescriptor{
		Label: req.Label + ":staging",
		Size:  outBytes,
		Usage: wgpu.BufferUsageMapRead | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return nil, fmt.Errorf("create staging buffer: %w", err)
	}
	defer staging.Release()

	if err := q.WriteBuffer(uni, 0, req.Uniform); err != nil {
		return nil, fmt.Errorf("write uniform: %w", err)
	}

	bg, err := dev.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: req.BindGroupLayout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: req.UniformBinding, Buffer: uni, Size: uint64(len(req.Uniform))},
			{Binding: req.OutputBinding, Buffer: out, Size: outBytes},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create bind group: %w", err)
	}
	defer bg.Release()

	enc, err := dev.CreateCommandEncoder(nil)
	if err != nil {
		return nil, fmt.Errorf("create command encoder: %w", err)
	}
	pass, err := enc.BeginComputePass(nil)
	if err != nil {
		return nil, fmt.Errorf("begin compute pass: %w", err)
	}
	pass.SetPipeline(req.Pipeline)
	pass.SetBindGroup(0, bg, nil)
	pass.Dispatch((uint32(req.Width)+req.WorkgroupX-1)/req.WorkgroupX, (uint32(req.Height)+req.WorkgroupY-1)/req.WorkgroupY, 1)
	if err := pass.End(); err != nil {
		return nil, fmt.Errorf("end compute pass: %w", err)
	}
	enc.CopyBufferToBuffer(out, 0, staging, 0, outBytes)
	cmd, err := enc.Finish()
	if err != nil {
		return nil, fmt.Errorf("finish: %w", err)
	}
	if _, err := q.Submit(cmd); err != nil {
		cmd.Release()
		return nil, fmt.Errorf("submit: %w", err)
	}

	// Buffer.Map starts an unpinned goroutine for Poll(PollWait). On Metal,
	// WaitIdle creates/drains a thread-local autorelease pool; migration of
	// that goroutine can crash in objc pool cleanup. Keep the entire mapping
	// lifecycle on the caller's locked Executor thread, including GPU wait.
	pending, err := staging.MapAsync(wgpu.MapModeRead, 0, outBytes)
	if err != nil {
		return nil, fmt.Errorf("map staging: %w", err)
	}
	dev.Poll(wgpu.PollWait)
	err = pending.Wait(context.Background())
	pending.Release()
	if err != nil {
		return nil, fmt.Errorf("map staging: %w", err)
	}
	defer staging.Unmap()
	rng, err := staging.MappedRange(0, outBytes)
	if err != nil {
		return nil, fmt.Errorf("mapped range: %w", err)
	}
	defer rng.Release()

	img := image.NewNRGBA(image.Rect(0, 0, req.Width, req.Height))
	copy(img.Pix, rng.Bytes())
	return img, nil
}
