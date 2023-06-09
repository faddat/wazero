package wazevoapi

import "github.com/tetratelabs/wazero/internal/wasm"

// OffsetData allows the frontend to get the information about offsets to the fields of wazevo.moduleContextOpaque and wazevo.executionContext,
// which are necessary for compiling various instructions.
//
// This should be unique per-Wasm module.
type OffsetData struct {
	// wazevo.executionContext
	ExecutionContextTrapCodeOffset Offset

	// TODO: add others later.
}

type Offset int32

func (o Offset) U32() uint32 {
	return uint32(o)
}

func NewOffsetData(m *wasm.Module) OffsetData {
	return OffsetData{ExecutionContextTrapCodeOffset: 0}
}
