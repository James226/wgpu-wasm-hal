//go:build js && wasm

package wasm

import (
	"syscall/js"

	"github.com/gogpu/wgpu/hal"
)

// Queue implements hal.Queue for the noop backend.
type Queue struct {
	queue js.Value
}

// Submit simulates command buffer submission.
// If a fence is provided, it is signaled with the given value.
func (q *Queue) Submit(commands []hal.CommandBuffer, fence hal.Fence, fenceValue uint64) error {
	if fence != nil {
		if f, ok := fence.(*Fence); ok {
			f.value.Store(fenceValue)
		}
	}

	c := make([]js.Value, len(commands))
	for i, cmd := range commands {
		c[i] = cmd.(*Resource).value
	}
	q.queue.Call("submit", unpackArray(c))

	return nil
}

// ReadBuffer reads data from a buffer.
func (q *Queue) ReadBuffer(buffer hal.Buffer, offset uint64, data []byte) error {
	complete := make(chan struct{})
	mapped := buffer.(*Resource).value.Call("mapAsync", uint64(1), offset, uint64(len(data)))
	mapped.Call("then", js.FuncOf(func(this js.Value, _ []js.Value) any {
		mappedRange := buffer.(*Resource).value.Call("getMappedRange", offset, uint64(len(data)))
		array := js.Global().Get("Uint8Array").New(mappedRange)
		js.CopyBytesToGo(data, array)
		buffer.(*Resource).value.Call("unmap")
		complete <- struct{}{}
		return nil
	}))
	<-complete
	// if b, ok := buffer.(*Buffer); ok && b.data != nil {
	// 	copy(data, b.data[offset:])
	// 	return nil
	// }
	return nil
}

func unpackArray[S ~[]E, E any](s S) []any {
	r := make([]any, len(s))
	for i, e := range s {
		r[i] = e
	}
	return r
}

// WriteBuffer simulates immediate buffer writes.
// If the buffer has storage, copies data to it.
func (q *Queue) WriteBuffer(buffer hal.Buffer, offset uint64, data []byte) error {
	jsArray := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(jsArray, data)
	q.queue.Call("writeBuffer", buffer.(*Resource).value, offset, jsArray, 0, uint64(len(data)))
	return nil
}

// WriteTexture simulates immediate texture writes.
// This is a no-op since textures don't store data.
func (q *Queue) WriteTexture(_ *hal.ImageCopyTexture, _ []byte, _ *hal.ImageDataLayout, _ *hal.Extent3D) error {
	return nil
}

// Present simulates surface presentation.
// Always succeeds.
func (q *Queue) Present(_ hal.Surface, _ hal.SurfaceTexture) error {
	return nil
}

// GetTimestampPeriod returns 1.0 nanosecond timestamp period.
func (q *Queue) GetTimestampPeriod() float32 {
	return 1.0
}
