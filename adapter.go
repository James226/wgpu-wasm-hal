//go:build js && wasm

package wasm

import (
	"syscall/js"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu/hal"
)

// Adapter implements hal.Adapter for the noop backend.
type Adapter struct {
	instance *Instance
	value    js.Value
}

func (a *Adapter) RequestAdapter() <-chan AdapterResult {
	resultChan := make(chan AdapterResult)
	promise := a.instance.gpu.Call("requestAdapter")

	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultChan <- AdapterResult{Adapter: &Adapter{value: args[0]}}
		return nil
	}))

	promise.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultChan <- AdapterResult{Error: js.Error{Value: args[0]}}
		return nil
	}))

	return resultChan
}

type DeviceResult struct {
	Device *Device
	Error  error
}

func (a *Adapter) RequestDevice() <-chan DeviceResult {
	resultChan := make(chan DeviceResult)
	promise := a.value.Call("requestDevice")

	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultChan <- DeviceResult{Device: &Device{device: args[0]}}
		return nil
	}))

	promise.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultChan <- DeviceResult{Error: js.Error{Value: args[0]}}
		return nil
	}))

	return resultChan
}

// Open creates a noop device with the requested features and limits.
// Always succeeds and returns a device/queue pair.
func (a *Adapter) Open(_ gputypes.Features, _ gputypes.Limits) (hal.OpenDevice, error) {
	result := <-a.RequestAdapter()
	if result.Error != nil {
		return hal.OpenDevice{}, result.Error
	}

	dr := <-result.Adapter.RequestDevice()
	if dr.Error != nil {
		return hal.OpenDevice{}, dr.Error
	}
	return hal.OpenDevice{
		Device: dr.Device,
		Queue:  &Queue{queue: dr.Device.device.Get("queue")},
	}, nil
}

// TextureFormatCapabilities returns default capabilities for all formats.
func (a *Adapter) TextureFormatCapabilities(_ gputypes.TextureFormat) hal.TextureFormatCapabilities {
	return hal.TextureFormatCapabilities{
		Flags: hal.TextureFormatCapabilitySampled |
			hal.TextureFormatCapabilityStorage |
			hal.TextureFormatCapabilityStorageReadWrite |
			hal.TextureFormatCapabilityRenderAttachment |
			hal.TextureFormatCapabilityBlendable |
			hal.TextureFormatCapabilityMultisample |
			hal.TextureFormatCapabilityMultisampleResolve,
	}
}

// SurfaceCapabilities returns default surface capabilities.
func (a *Adapter) SurfaceCapabilities(_ hal.Surface) *hal.SurfaceCapabilities {
	return &hal.SurfaceCapabilities{
		Formats: []gputypes.TextureFormat{
			gputypes.TextureFormatBGRA8Unorm,
			gputypes.TextureFormatRGBA8Unorm,
		},
		PresentModes: []hal.PresentMode{
			hal.PresentModeImmediate,
			hal.PresentModeMailbox,
			hal.PresentModeFifo,
			hal.PresentModeFifoRelaxed,
		},
		AlphaModes: []hal.CompositeAlphaMode{
			hal.CompositeAlphaModeOpaque,
			hal.CompositeAlphaModePremultiplied,
			hal.CompositeAlphaModeUnpremultiplied,
			hal.CompositeAlphaModeInherit,
		},
	}
}

// Destroy is a no-op for the noop adapter.
func (a *Adapter) Destroy() {}
