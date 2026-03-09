//go:build js && wasm

// Copyright 2025 The GoGPU Authors
// SPDX-License-Identifier: MIT

// Command compute-sum demonstrates a parallel reduction (sum) using a GPU
// compute shader. It uploads an array of uint32 values to the GPU, dispatches
// a compute shader that sums contiguous pairs, and reads back the partial
// results. The final summation is performed on the CPU.
//
// The example is headless (no window required) and works on any supported GPU.
package main

import (
	"fmt"
	"log"
	"syscall/js"
	"unsafe"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"

	// Register all available GPU backends (Vulkan, DX12, GLES, Metal, etc.)
	_ "github.com/james226/wgpu-wasm-hal"
)

// sumShaderWGSL performs pairwise addition: output[i] = input[2*i] + input[2*i+1].
// Each workgroup thread handles one output element.
const sumShaderWGSL = `
struct VertexOut {
  @builtin(position) position : vec4f,
  @location(0) color : vec4f
}

@vertex
fn vertex_main(@location(0) position: vec4f,
               @location(1) color: vec4f) -> VertexOut
{
  var output : VertexOut;
  output.position = position;
  output.color = color;
  return output;
}

@fragment
fn fragment_main(fragData: VertexOut) -> @location(0) vec4f
{
  return fragData.color;
}
`

func main() {
	if err := run(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}
}

func run() error {
	fmt.Println("=== Compute Shader: Parallel Sum ===")
	fmt.Println()

	device, surface, cleanup, err := initDevice()
	if err != nil {
		return err
	}
	defer cleanup()

	vertexData := prepareInput()

	bufs, err := createBuffers(device, vertexData)
	if err != nil {
		return err
	}
	defer bufs.release()

	ps, err := createPipeline(device, bufs)
	if err != nil {
		return err
	}
	defer ps.release()

	err = dispatch(device, surface, ps, bufs)
	if err != nil {
		return err
	}

	return nil
}

func initDevice() (*wgpu.Device, *wgpu.Surface, func(), error) {
	fmt.Print("1. Creating instance... ")
	instance, err := wgpu.CreateInstance(nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("CreateInstance: %w", err)
	}
	fmt.Println("OK")

	fmt.Print("2. Requesting adapter... ")
	adapter, err := instance.RequestAdapter(nil)
	if err != nil {
		instance.Release()
		return nil, nil, nil, fmt.Errorf("RequestAdapter: %w", err)
	}
	fmt.Printf("OK (%s)\n", adapter.Info().Name)

	fmt.Print("3. Creating device... ")
	device, err := adapter.RequestDevice(nil)
	if err != nil {
		adapter.Release()
		instance.Release()
		return nil, nil, nil, fmt.Errorf("RequestDevice: %w", err)
	}

	canvas := js.Global().Get("document").Call("getElementById", "canvas")

	surface, err := instance.CreateSurface(uintptr(unsafe.Pointer(&canvas)), 0)
	if err != nil {
		instance.Release()
		return nil, nil, nil, fmt.Errorf("CreateSurface: %w", err)
	}
	fmt.Println("OK")

	surface.Configure(device, &wgpu.SurfaceConfiguration{
		Usage:       wgpu.TextureUsageRenderAttachment,
		Format:      gputypes.TextureFormatBGRA8Unorm,
		Width:       640,
		Height:      480,
		PresentMode: wgpu.PresentModeFifo,
	})

	cleanup := func() {
		device.Release()
		adapter.Release()
		surface.Release()
		instance.Release()
	}
	return device, surface, cleanup, nil
}

func prepareInput() []float32 {
	vertexData := []float32{
		0.0, 0.6, 0, 1, 1, 0, 0, 1, -0.5, -0.6, 0, 1, 0, 1, 0, 1, 0.5, -0.6, 0, 1, 0,
		0, 1, 1,
	}
	return vertexData
}

type bufferSet struct {
	input *wgpu.Buffer
}

func (b *bufferSet) release() {
	b.input.Release()
}

const SIZE_FLOAT32 = int(unsafe.Sizeof(float32(0)))

func createBuffers(device *wgpu.Device, vertexData []float32) (*bufferSet, error) {
	fmt.Print("5. Creating buffers... ")
	vertexBuf, err := device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "vertex", Size: uint64(len(vertexData) * 4),
		Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return nil, fmt.Errorf("create vertex buffer: %w", err)
	}
	data := unsafe.Slice((*byte)(unsafe.Pointer(&vertexData[0])), len(vertexData)*SIZE_FLOAT32)
	if err := device.Queue().WriteBuffer(vertexBuf, 0, data); err != nil {
		return nil, fmt.Errorf("write vertex buffer: %w", err)
	}
	fmt.Println("OK")

	return &bufferSet{input: vertexBuf}, nil
}

type pipelineSet struct {
	shader   interface{ Release() }
	pipeline *wgpu.RenderPipeline
}

func (p *pipelineSet) release() {
	p.pipeline.Release()
	p.shader.Release()
}

func createPipeline(device *wgpu.Device, bufs *bufferSet) (*pipelineSet, error) {
	fmt.Print("6. Creating render pipeline... ")
	shader, err := device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label: "render-shader", WGSL: sumShaderWGSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create shader: %w", err)
	}
	pipeline, err := device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "render-pipeline",
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vertex_main",
			Buffers: []wgpu.VertexBufferLayout{
				{
					ArrayStride: 8 * 4,
					Attributes: []gputypes.VertexAttribute{
						{ShaderLocation: 0, Format: gputypes.VertexFormatFloat32x4, Offset: 0},
						{ShaderLocation: 1, Format: gputypes.VertexFormatFloat32x4, Offset: 4 * 4},
					},
					StepMode: gputypes.VertexStepModeVertex,
				},
			},
		},
		Fragment: &wgpu.FragmentState{
			Module: shader, EntryPoint: "fragment_main",
			Targets: []wgpu.ColorTargetState{
				{Format: gputypes.TextureFormatBGRA8Unorm},
			},
		},
		Primitive: wgpu.PrimitiveState{
			Topology: gputypes.PrimitiveTopologyTriangleList,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create pipeline: %w", err)
	}
	fmt.Println("OK")

	return &pipelineSet{
		shader:   shader,
		pipeline: pipeline,
	}, nil
}

func dispatch(device *wgpu.Device, surface *wgpu.Surface, ps *pipelineSet, bufs *bufferSet) error {
	fmt.Print("7. Dispatching render... ")
	encoder, err := device.CreateCommandEncoder(nil)
	if err != nil {
		return fmt.Errorf("create encoder: %w", err)
	}
	texture, _, err := surface.GetCurrentTexture()
	if err != nil {
		return fmt.Errorf("get current texture: %w", err)
	}
	view, err := texture.CreateView(nil)
	if err != nil {
		return fmt.Errorf("create texture view: %w", err)
	}
	pass, err := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:       view,
				LoadOp:     gputypes.LoadOpClear,
				StoreOp:    gputypes.StoreOpStore,
				ClearValue: gputypes.Color{R: 0, G: 0.5, B: 1, A: 1},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("begin render pass: %w", err)
	}
	pass.SetPipeline(ps.pipeline)
	pass.SetVertexBuffer(0, bufs.input, 0)
	pass.Draw(3, 1, 0, 0)
	if err := pass.End(); err != nil {
		return fmt.Errorf("end render pass: %w", err)
	}
	cmdBuf, err := encoder.Finish()
	if err != nil {
		return fmt.Errorf("finish encoder: %w", err)
	}
	if err := device.Queue().Submit(cmdBuf); err != nil {
		return fmt.Errorf("submit: %w", err)
	}
	fmt.Println("OK")
	return nil
}
