//go:build !js

package main

// Native builds register the platform HAL backends (Metal, Vulkan, DX12,
// software). The browser build needs none of this: gogpu/wgpu talks to the
// browser's WebGPU API directly through syscall/js.
import _ "github.com/gogpu/wgpu/hal/allbackends"
