package backend_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/isa/arm64"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/frontend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/testcases"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/wazevoapi"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/internal/wasm"
)

func newMachine() backend.Machine {
	switch runtime.GOARCH {
	case "arm64":
		return arm64.NewBackend()
	default:
		panic("unsupported architecture")
	}
}

func TestE2E(t *testing.T) {
	type testCase struct {
		name          string
		m             *wasm.Module
		targetIndex   uint32
		afterLowering string
	}

	for _, tc := range []testCase{
		{name: "empty", m: testcases.Empty.Module, targetIndex: 0, afterLowering: `
L1 (SSA Block: blk0):
	ret
`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			od := wazevoapi.NewOffsetData(tc.m)
			ssab := ssa.NewBuilder()
			fc := frontend.NewFrontendCompiler(od, tc.m, ssab)
			be := backend.NewBackendCompiler(newMachine(), ssab)

			// Lowers the Wasm to SSA.
			typeIndex := tc.m.FunctionSection[tc.targetIndex]
			code := &tc.m.CodeSection[tc.targetIndex]
			fc.Init(tc.targetIndex, &tc.m.TypeSection[typeIndex], code.LocalTypes, code.Body)
			err := fc.LowerToSSA()
			require.NoError(t, err)

			// Need to run passes before lowering to machine code.
			ssab.RunPasses()
			ssab.LayoutBlocks()

			// Lowers the SSA to ISA specific code.
			backend.ExportLower(be)

			fmt.Println("============ lowering result ============")
			fmt.Println(be.Format())

			require.Equal(t, tc.afterLowering, be.Format())
		})
	}
}
