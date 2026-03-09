//go:build js && wasm

package wasm

import (
	"strings"
	"syscall/js"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu/hal"
)

// CommandEncoder implements hal.CommandEncoder for the noop backend.
type CommandEncoder struct {
	value js.Value
}

// BeginEncoding is a no-op.
func (c *CommandEncoder) BeginEncoding(_ string) error {
	return nil
}

// EndEncoding returns a placeholder command buffer.
func (c *CommandEncoder) EndEncoding() (hal.CommandBuffer, error) {
	r := c.value.Call("finish")
	return &Resource{value: r}, nil
}

// DiscardEncoding is a no-op.
func (c *CommandEncoder) DiscardEncoding() {}

// ResetAll is a no-op.
func (c *CommandEncoder) ResetAll(_ []hal.CommandBuffer) {}

// TransitionBuffers is a no-op.
func (c *CommandEncoder) TransitionBuffers(_ []hal.BufferBarrier) {}

// TransitionTextures is a no-op.
func (c *CommandEncoder) TransitionTextures(_ []hal.TextureBarrier) {}

// ClearBuffer is a no-op.
func (c *CommandEncoder) ClearBuffer(_ hal.Buffer, _, _ uint64) {}

// CopyBufferToBuffer is a no-op.
func (c *CommandEncoder) CopyBufferToBuffer(source, destination hal.Buffer, params []hal.BufferCopy) {
	c.value.Call("copyBufferToBuffer", source.(*Resource).value, destination.(*Resource).value, params[0].Size)
}

// CopyBufferToTexture is a no-op.
func (c *CommandEncoder) CopyBufferToTexture(_ hal.Buffer, _ hal.Texture, _ []hal.BufferTextureCopy) {
}

// CopyTextureToBuffer is a no-op.
func (c *CommandEncoder) CopyTextureToBuffer(_ hal.Texture, _ hal.Buffer, _ []hal.BufferTextureCopy) {
}

// CopyTextureToTexture is a no-op.
func (c *CommandEncoder) CopyTextureToTexture(_, _ hal.Texture, _ []hal.TextureCopy) {}

// ResolveQuerySet is a no-op.
func (c *CommandEncoder) ResolveQuerySet(_ hal.QuerySet, _, _ uint32, _ hal.Buffer, _ uint64) {}

// BeginRenderPass returns a noop render pass encoder.
func (c *CommandEncoder) BeginRenderPass(desc *hal.RenderPassDescriptor) hal.RenderPassEncoder {
	descriptor := make(map[string]any)
	descriptor["colorAttachments"] = func() []any {
		attachments := make([]any, len(desc.ColorAttachments))
		for i, a := range desc.ColorAttachments {
			attachments[i] = map[string]any{
				"view":       a.View.(*Resource).value,
				"loadOp":     strings.ToLower(a.LoadOp.String()),
				"storeOp":    strings.ToLower(a.StoreOp.String()),
				"clearValue": map[string]any{"r": a.ClearValue.R, "g": a.ClearValue.G, "b": a.ClearValue.B, "a": a.ClearValue.A},
			}
		}
		return attachments
	}()
	if desc.DepthStencilAttachment != nil {
		descriptor["depthStencilAttachment"] = map[string]any{
			"view":              desc.DepthStencilAttachment.View.(*Resource).value,
			"depthLoadOp":       strings.ToLower(desc.DepthStencilAttachment.DepthLoadOp.String()),
			"depthStoreOp":      strings.ToLower(desc.DepthStencilAttachment.DepthStoreOp.String()),
			"depthClearValue":   desc.DepthStencilAttachment.DepthClearValue,
			"stencilLoadOp":     strings.ToLower(desc.DepthStencilAttachment.StencilLoadOp.String()),
			"stencilStoreOp":    strings.ToLower(desc.DepthStencilAttachment.StencilStoreOp.String()),
			"stencilClearValue": desc.DepthStencilAttachment.StencilClearValue,
		}
	}
	pass := c.value.Call("beginRenderPass", descriptor)
	return &RenderPassEncoder{value: pass}
}

// BeginComputePass returns a noop compute pass encoder.
func (c *CommandEncoder) BeginComputePass(_ *hal.ComputePassDescriptor) hal.ComputePassEncoder {
	p := c.value.Call("beginComputePass")
	return &ComputePassEncoder{value: p}
}

// RenderPassEncoder implements hal.RenderPassEncoder for the noop backend.
type RenderPassEncoder struct {
	value js.Value
}

// End is a no-op.
func (r *RenderPassEncoder) End() {
	r.value.Call("end")
}

// SetPipeline is a no-op.
func (c *RenderPassEncoder) SetPipeline(pipeline hal.RenderPipeline) {
	c.value.Call("setPipeline", pipeline.(*Resource).value)
}

// SetBindGroup is a no-op.
func (r *RenderPassEncoder) SetBindGroup(_ uint32, _ hal.BindGroup, _ []uint32) {}

// SetVertexBuffer is a no-op.
func (r *RenderPassEncoder) SetVertexBuffer(slot uint32, buffer hal.Buffer, offset uint64) {
	r.value.Call("setVertexBuffer", slot, buffer.(*Resource).value, offset)
}

// SetIndexBuffer is a no-op.
func (r *RenderPassEncoder) SetIndexBuffer(_ hal.Buffer, _ gputypes.IndexFormat, _ uint64) {}

// SetViewport is a no-op.
func (r *RenderPassEncoder) SetViewport(_, _, _, _, _, _ float32) {}

// SetScissorRect is a no-op.
func (r *RenderPassEncoder) SetScissorRect(_, _, _, _ uint32) {}

// SetBlendConstant is a no-op.
func (r *RenderPassEncoder) SetBlendConstant(_ *gputypes.Color) {}

// SetStencilReference is a no-op.
func (r *RenderPassEncoder) SetStencilReference(_ uint32) {}

// Draw is a no-op.
func (r *RenderPassEncoder) Draw(vertexCount, instanceCount, firstVertex, firstInstance uint32) {
	r.value.Call("draw", vertexCount, instanceCount, firstVertex, firstInstance)
}

// DrawIndexed is a no-op.
func (r *RenderPassEncoder) DrawIndexed(_, _, _ uint32, _ int32, _ uint32) {}

// DrawIndirect is a no-op.
func (r *RenderPassEncoder) DrawIndirect(_ hal.Buffer, _ uint64) {}

// DrawIndexedIndirect is a no-op.
func (r *RenderPassEncoder) DrawIndexedIndirect(_ hal.Buffer, _ uint64) {}

// ExecuteBundle is a no-op.
func (r *RenderPassEncoder) ExecuteBundle(_ hal.RenderBundle) {}

// ComputePassEncoder implements hal.ComputePassEncoder for the noop backend.
type ComputePassEncoder struct {
	value js.Value
}

// End is a no-op.
func (c *ComputePassEncoder) End() {
	c.value.Call("end")
}

// SetPipeline is a no-op.
func (c *ComputePassEncoder) SetPipeline(pipeline hal.ComputePipeline) {
	c.value.Call("setPipeline", pipeline.(*Resource).value)
}

// SetBindGroup is a no-op.
func (c *ComputePassEncoder) SetBindGroup(idx uint32, bindGroup hal.BindGroup, _ []uint32) {
	c.value.Call("setBindGroup", idx, bindGroup.(*Resource).value)
}

// Dispatch is a no-op.
func (c *ComputePassEncoder) Dispatch(x, y, z uint32) {
	c.value.Call("dispatchWorkgroups", x, y, z)
}

// DispatchIndirect is a no-op.
func (c *ComputePassEncoder) DispatchIndirect(_ hal.Buffer, _ uint64) {}
