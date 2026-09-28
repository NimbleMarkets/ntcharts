package main

import (
	"testing"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
)

func TestAdapterUsable(t *testing.T) {
	cases := []struct {
		name    string
		browser bool
		info    wgpu.AdapterInfo
		wantErr bool
	}{
		{"native discrete", false, wgpu.AdapterInfo{Name: "M3", Backend: gputypes.BackendMetal, DeviceType: gputypes.DeviceTypeIntegratedGPU}, false},
		{"native cpu rasterizer", false, wgpu.AdapterInfo{Name: "software", Backend: gputypes.BackendVulkan, DeviceType: gputypes.DeviceTypeCPU}, true},
		{"native no backend", false, wgpu.AdapterInfo{Name: "noop"}, true},
		// The browser exposes no backend or device type, only a name.
		{"browser minimal info", true, wgpu.AdapterInfo{Name: "WebGPU Adapter"}, false},
		{"browser cpu", true, wgpu.AdapterInfo{Name: "swiftshader", DeviceType: gputypes.DeviceTypeCPU}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := adapterUsable(c.info, c.browser)
			if (err != nil) != c.wantErr {
				t.Fatalf("adapterUsable(%+v, browser=%v) = %v, wantErr %v", c.info, c.browser, err, c.wantErr)
			}
		})
	}
}
