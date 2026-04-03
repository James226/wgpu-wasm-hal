//go:build js && wasm

package wasm

import (
	"fmt"
	"strings"
	"syscall/js"
	"time"
	"unsafe"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu/hal"
)

// Device implements hal.Device for the noop backend.
type Device struct {
	device js.Value
}

// CreateBuffer creates a noop buffer.
// Optionally stores data if MappedAtCreation is true.
func (d *Device) CreateBuffer(desc *hal.BufferDescriptor) (hal.Buffer, error) {
	buffer := d.device.Call("createBuffer", map[string]any{
		"label":            desc.Label,
		"size":             desc.Size,
		"usage":            uint64(desc.Usage),
		"mappedAtCreation": desc.MappedAtCreation,
	})

	return &Resource{
		value: buffer,
	}, nil
}

// DestroyBuffer is a no-op.
func (d *Device) DestroyBuffer(buffer hal.Buffer) {
	buffer.(*Resource).value.Call("destroy")
}

// CreateTexture creates a noop texture.
func (d *Device) CreateTexture(desc *hal.TextureDescriptor) (hal.Texture, error) {
	texture := d.device.Call("createTexture", map[string]any{
		"label": desc.Label,
		"size": map[string]any{
			"width":  desc.Size.Width,
			"height": desc.Size.Height,
		},
		"format": strings.Replace(strings.ToLower(desc.Format.String()), "stencil8", "-stencil8", -1),
		"usage":  uint64(desc.Usage),
	})
	return &Texture{Resource: Resource{value: texture}}, nil
}

// DestroyTexture is a no-op.
func (d *Device) DestroyTexture(texture hal.Texture) {
	texture.(*Texture).value.Call("destroy")
}

// CreateTextureView creates a noop texture view.
func (d *Device) CreateTextureView(texture hal.Texture, _ *hal.TextureViewDescriptor) (hal.TextureView, error) {
	surface, ok := texture.(*SurfaceTexture)
	if ok {
		return &Resource{value: surface.Texture.value.Call("createView")}, nil
	}
	v := texture.(*Texture).value.Call("createView")
	return &Resource{value: v}, nil
}

// DestroyTextureView is a no-op.
func (d *Device) DestroyTextureView(texture hal.TextureView) {
	texture.(*Resource).value.Call("destroy")
}

// CreateSampler creates a noop sampler.
func (d *Device) CreateSampler(_ *hal.SamplerDescriptor) (hal.Sampler, error) {
	return &Resource{}, nil
}

// DestroySampler is a no-op.
func (d *Device) DestroySampler(_ hal.Sampler) {}

func BufferBindingTypeToJs(t gputypes.BufferBindingType) string {
	switch t {
	case gputypes.BufferBindingTypeUndefined:
		return "undefined"
	case gputypes.BufferBindingTypeUniform:
		return "uniform"
	case gputypes.BufferBindingTypeStorage:
		return "storage"
	case gputypes.BufferBindingTypeReadOnlyStorage:
		return "read-only-storage"
	default:
		return "unknown"
	}
}

func SamplerBindingTypeToJs(t gputypes.SamplerBindingType) string {
	switch t {
	case gputypes.SamplerBindingTypeUndefined:
		return "undefined"
	case gputypes.SamplerBindingTypeFiltering:
		return "filtering"
	case gputypes.SamplerBindingTypeNonFiltering:
		return "non-filtering"
	case gputypes.SamplerBindingTypeComparison:
		return "comparison"
	default:
		return "unknown"
	}
}

func (d *Device) CreateBindGroupLayout(desc *hal.BindGroupLayoutDescriptor) (hal.BindGroupLayout, error) {
	descriptor := map[string]any{
		"label": desc.Label,
		"entries": func() []any {
			entries := make([]any, len(desc.Entries))
			for i, entry := range desc.Entries {
				e := make(map[string]any)
				e["binding"] = entry.Binding
				e["visibility"] = uint32(entry.Visibility)
				if entry.Buffer != nil {
					e["buffer"] = map[string]any{
						"type": BufferBindingTypeToJs(entry.Buffer.Type),
					}
				}
				if entry.Sampler != nil {
					e["sampler"] = map[string]any{
						"type": SamplerBindingTypeToJs(entry.Sampler.Type),
					}
				}
				entries[i] = e
			}
			return entries
		}(),
	}

	bgl := d.device.Call("createBindGroupLayout", descriptor)
	return &Resource{value: bgl}, nil
}

// DestroyBindGroupLayout is a no-op.
func (d *Device) DestroyBindGroupLayout(_ hal.BindGroupLayout) {}

// CreateBindGroup creates a noop bind group.
func (d *Device) CreateBindGroup(desc *hal.BindGroupDescriptor) (hal.BindGroup, error) {
	descriptor := map[string]any{
		"label":  desc.Label,
		"layout": desc.Layout.(*Resource).value,
		"entries": func() []any {
			entries := make([]any, len(desc.Entries))
			for i, entry := range desc.Entries {
				entries[i] = map[string]any{
					"binding": entry.Binding,
					"resource": func() any {
						switch r := entry.Resource.(type) {
						case gputypes.BufferBinding:
							return map[string]any{
								"buffer": *(*js.Value)(unsafe.Pointer(r.Buffer)),
							}
						}
						return nil
					}(),
				}
			}
			return entries
		}(),
	}

	bg := d.device.Call("createBindGroup", descriptor)
	return &Resource{value: bg}, nil
}

// DestroyBindGroup is a no-op.
func (d *Device) DestroyBindGroup(_ hal.BindGroup) {}

// CreatePipelineLayout creates a noop pipeline layout.
func (d *Device) CreatePipelineLayout(desc *hal.PipelineLayoutDescriptor) (hal.PipelineLayout, error) {
	descriptor := map[string]any{
		"label": desc.Label,
		"bindGroupLayouts": func() []any {
			layouts := make([]any, len(desc.BindGroupLayouts))
			for i, layout := range desc.BindGroupLayouts {
				layouts[i] = layout.(*Resource).value
			}
			return layouts
		}(),
	}

	pl := d.device.Call("createPipelineLayout", descriptor)
	return &Resource{value: pl}, nil
}

// DestroyPipelineLayout is a no-op.
func (d *Device) DestroyPipelineLayout(_ hal.PipelineLayout) {}

// CreateShaderModule creates a noop shader module.
func (d *Device) CreateShaderModule(desc *hal.ShaderModuleDescriptor) (hal.ShaderModule, error) {
	descriptor := map[string]any{
		"label": desc.Label,
		"code":  desc.Source.WGSL,
	}
	shaderModule := d.device.Call("createShaderModule", descriptor)
	return &Resource{value: shaderModule}, nil
}

// DestroyShaderModule is a no-op.
func (d *Device) DestroyShaderModule(_ hal.ShaderModule) {}

func VertexStepModeToJs(m gputypes.VertexStepMode) string {
	switch m {
	case gputypes.VertexStepModeUndefined:
		return "undefined"
	case gputypes.VertexStepModeVertexBufferNotUsed:
		return "vertex-buffer-not-used"
	case gputypes.VertexStepModeVertex:
		return "vertex"
	case gputypes.VertexStepModeInstance:
		return "instance"
	default:
		return "unknown"
	}
}

func PrimitiveTopologyToJs(t gputypes.PrimitiveTopology) string {
	switch t {
	case gputypes.PrimitiveTopologyUndefined:
		return "undefined"
	case gputypes.PrimitiveTopologyPointList:
		return "point-list"
	case gputypes.PrimitiveTopologyLineList:
		return "line-list"
	case gputypes.PrimitiveTopologyLineStrip:
		return "line-strip"
	case gputypes.PrimitiveTopologyTriangleList:
		return "triangle-list"
	case gputypes.PrimitiveTopologyTriangleStrip:
		return "triangle-strip"
	default:
		return "unknown"
	}
}

// CreateRenderPipeline creates a noop render pipeline.
func (d *Device) CreateRenderPipeline(desc *hal.RenderPipelineDescriptor) (hal.RenderPipeline, error) {
	descriptor := map[string]any{
		"label": desc.Label,
		"layout": func() any {
			if desc.Layout != nil {
				return desc.Layout.(*Resource).value
			}
			return "auto"
		}(),
		"vertex": map[string]any{
			"module":     desc.Vertex.Module.(*Resource).value,
			"entryPoint": desc.Vertex.EntryPoint,
			"buffers": func() []any {
				buffers := make([]any, len(desc.Vertex.Buffers))
				for i, vb := range desc.Vertex.Buffers {
					buffers[i] = map[string]any{
						"arrayStride": vb.ArrayStride,
						"stepMode":    VertexStepModeToJs(vb.StepMode),
						"attributes": func() []any {
							attributes := make([]any, len(vb.Attributes))
							for j, attr := range vb.Attributes {
								attributes[j] = map[string]any{
									"shaderLocation": attr.ShaderLocation,
									"format":         strings.ToLower(attr.Format.String()),
									"offset":         attr.Offset,
								}
							}
							return attributes
						}(),
					}
				}
				return buffers
			}(),
		},
		"fragment": func() any {
			if desc.Fragment != nil {
				return map[string]any{
					"module":     desc.Fragment.Module.(*Resource).value,
					"entryPoint": desc.Fragment.EntryPoint,
					"targets": func() []any {
						targets := make([]any, len(desc.Fragment.Targets))
						for i, target := range desc.Fragment.Targets {
							targets[i] = map[string]any{
								"format": strings.ToLower(target.Format.String()),
							}
						}
						return targets
					}(),
				}
			}
			return nil
		}(),
		"primitive": map[string]any{
			"topology": PrimitiveTopologyToJs(desc.Primitive.Topology),
		},
	}
	rp := d.device.Call("createRenderPipeline", descriptor)
	return &Resource{value: rp}, nil
}

// DestroyRenderPipeline is a no-op.
func (d *Device) DestroyRenderPipeline(_ hal.RenderPipeline) {}

// CreateComputePipeline creates a noop compute pipeline.
func (d *Device) CreateComputePipeline(desc *hal.ComputePipelineDescriptor) (hal.ComputePipeline, error) {
	descriptor := map[string]any{
		"label":  desc.Label,
		"layout": desc.Layout.(*Resource).value,
		"compute": map[string]any{
			"module":     desc.Compute.Module.(*Resource).value,
			"entryPoint": desc.Compute.EntryPoint,
		},
	}
	cp := d.device.Call("createComputePipeline", descriptor)
	return &Resource{value: cp}, nil
}

// DestroyComputePipeline is a no-op.
func (d *Device) DestroyComputePipeline(_ hal.ComputePipeline) {}

// CreateQuerySet returns ErrTimestampsNotSupported (noop backend has no GPU).
func (d *Device) CreateQuerySet(_ *hal.QuerySetDescriptor) (hal.QuerySet, error) {
	return nil, hal.ErrTimestampsNotSupported
}

// DestroyQuerySet is a no-op.
func (d *Device) DestroyQuerySet(_ hal.QuerySet) {}

// CreateCommandEncoder creates a noop command encoder.
func (d *Device) CreateCommandEncoder(desc *hal.CommandEncoderDescriptor) (hal.CommandEncoder, error) {
	descriptor := map[string]any{
		"label": desc.Label,
	}
	cp := d.device.Call("createCommandEncoder", descriptor)

	return &CommandEncoder{value: cp}, nil
}

// CreateFence creates a noop fence with atomic counter.
func (d *Device) CreateFence() (hal.Fence, error) {
	return &Fence{}, nil
}

// DestroyFence is a no-op.
func (d *Device) DestroyFence(_ hal.Fence) {}

// Wait simulates waiting for a fence value.
// Always returns true immediately (fence reached).
func (d *Device) Wait(fence hal.Fence, value uint64, _ time.Duration) (bool, error) {
	f, ok := fence.(*Fence)
	if !ok {
		return true, nil
	}
	// Check if fence has reached the value
	return f.value.Load() >= value, nil
}

// ResetFence resets a fence to the unsignaled state.
func (d *Device) ResetFence(fence hal.Fence) error {
	f, ok := fence.(*Fence)
	if !ok {
		return nil
	}
	f.value.Store(0)
	return nil
}

// GetFenceStatus returns true if the fence is signaled (non-blocking).
func (d *Device) GetFenceStatus(fence hal.Fence) (bool, error) {
	f, ok := fence.(*Fence)
	if !ok {
		return false, nil
	}
	return f.value.Load() > 0, nil
}

// FreeCommandBuffer is a no-op for the noop device.
func (d *Device) FreeCommandBuffer(cmdBuffer hal.CommandBuffer) {}

// CreateRenderBundleEncoder is a no-op for the noop device.
func (d *Device) CreateRenderBundleEncoder(desc *hal.RenderBundleEncoderDescriptor) (hal.RenderBundleEncoder, error) {
	return nil, fmt.Errorf("noop: render bundles not supported")
}

// DestroyRenderBundle is a no-op for the noop device.
func (d *Device) DestroyRenderBundle(bundle hal.RenderBundle) {}

// WaitIdle is a no-op for the noop device.
func (d *Device) WaitIdle() error { return nil }

func (d *Device) Destroy() {
	d.device.Call("destroy")
}

func (d *Device) ToJS() js.Value {
	return d.device
}
