package wazevoapi

import "github.com/tetratelabs/wazero/internal/wasm"

// OffsetData allows the frontend to get the information about offsets to the fields of wazevo.moduleContextOpaque and wazevo.executionContext,
// which are necessary for compiling various instructions.
//
// This should be unique per-Wasm module.
type OffsetData struct {
	// ExecutionContextTrapCodeOffset is an offset of `trapCode` field in wazevo.executionContext
	ExecutionContextTrapCodeOffset Offset
	// ExecutionContextCallerModuleContextPtr is an offset of `callerModuleContextPtr` field in wazevo.executionContext
	ExecutionContextCallerModuleContextPtr Offset

	// TODO: add others later.
}

// Offset represents an offset of a field of a struct.
type Offset int32

// U32 encodes an Offset as uint32 for convenience.
func (o Offset) U32() uint32 {
	return uint32(o)
}

// NewOffsetData creates a OffsetData for the given Module.
func NewOffsetData(_ *wasm.Module) OffsetData {
	return OffsetData{
		ExecutionContextTrapCodeOffset:         0,
		ExecutionContextCallerModuleContextPtr: 8,
	}
}
