//go:build js && wasm
// +build js,wasm

package wasm

import (
	"fmt"
	"syscall/js"
	"unsafe"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu/hal"
)

// API implements hal.Backend for the noop backend.
type API struct{}

// Variant returns the backend type identifier.
func (API) Variant() gputypes.Backend {
	return gputypes.BackendBrowserWebGPU
}

// CreateInstance creates a new noop instance.
// Always succeeds and returns a placeholder instance.
func (API) CreateInstance(_ *hal.InstanceDescriptor) (hal.Instance, error) {
	gpu := js.Global().Get("navigator").Get("gpu")
	if gpu.IsUndefined() {
		return nil, fmt.Errorf("WebGPU not supported")
	}
	return &Instance{gpu: gpu}, nil
}

// Instance implements hal.Instance for the noop backend.
type Instance struct {
	gpu js.Value
}

// CreateSurface creates a noop surface.
// Always succeeds regardless of display/window handles.
func (i *Instance) CreateSurface(displayHandle, _ uintptr) (hal.Surface, error) {
	canvas := *(*js.Value)(unsafe.Pointer(displayHandle))
	context := canvas.Call("getContext", "webgpu")

	return &Surface{canvas: canvas, context: context}, nil
}

type AdapterResult struct {
	Adapter *Adapter
	Error   error
}

// EnumerateAdapters returns a single default noop adapter.
// The surfaceHint is ignored.
func (i *Instance) EnumerateAdapters(_ hal.Surface) []hal.ExposedAdapter {
	return []hal.ExposedAdapter{
		{
			Adapter: &Adapter{instance: i},
			Info: gputypes.AdapterInfo{
				Name:       "WASM Adapter",
				Vendor:     "GoGPU",
				VendorID:   0,
				DeviceID:   0,
				DeviceType: gputypes.DeviceTypeOther,
				Driver:     "wasm-1.0",
				DriverInfo: "WASM WebGPU adapter",
				Backend:    gputypes.BackendBrowserWebGPU,
			},
			Features: 0, // No features supported
			Capabilities: hal.Capabilities{
				Limits: gputypes.DefaultLimits(),
				AlignmentsMask: hal.Alignments{
					BufferCopyOffset: 4,
					BufferCopyPitch:  256,
				},
				DownlevelCapabilities: hal.DownlevelCapabilities{
					ShaderModel: 0,
					Flags:       0,
				},
			},
		},
	}
}

// Destroy is a no-op for the noop instance.
func (i *Instance) Destroy() {}
