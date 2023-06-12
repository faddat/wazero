package wazevo

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/frontend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/wazevoapi"
	"github.com/tetratelabs/wazero/internal/filecache"
	"github.com/tetratelabs/wazero/internal/platform"
	"github.com/tetratelabs/wazero/internal/wasm"
)

type (
	// engine implements wasm.Engine.
	engine struct {
		compiledModules map[wasm.ModuleID]*compiledModule
		mux             sync.RWMutex
	}

	// compiledModule is a compiled variant of a wasm.Module and ready to be used for instantiation.
	compiledModule struct {
		executable        []byte
		compiledFunctions []compiledFunction
	}

	compiledFunction struct {
		offsetInExecutable int
	}

	// TODO:
	moduleContext struct{}
	// TODO:
	moduleContextOpaque struct{}
	// TODO:
	executionContext struct {
		// trapCode holds the wazevoapi.TrapCode if it happens.
		trapCode wazevoapi.TrapCode
		// callerModuleContextPtr holds the moduleContextOpaque for Go function calls.
		callerModuleContextPtr *byte
	}
)

var _ wasm.Engine = (*engine)(nil)

// NewEngine returns the implementation of wasm.Engine.
func NewEngine(_ context.Context, _ api.CoreFeatures, _ filecache.Cache) wasm.Engine {
	return &engine{compiledModules: make(map[wasm.ModuleID]*compiledModule)}
}

// CompileModule implements wasm.Engine.
func (e *engine) CompileModule(_ context.Context, module *wasm.Module, _ []experimental.FunctionListener, ensureTermination bool) error {
	cm := &compiledModule{}

	importedFns, localFns := int(module.ImportFunctionCount), len(module.FunctionSection)
	if importedFns+localFns == 0 {
		e.addCompiledModule(module, cm)
		return nil
	}

	offsets := wazevoapi.NewOffsetData(module)

	var totalSize int
	bodies := make([][]byte, localFns)

	ssaBuilder := ssa.NewBuilder()
	fe, be := frontend.NewFrontendCompiler(offsets, module, ssaBuilder), backend.NewBackendCompiler(ssaBuilder)
	for i := range module.CodeSection {
		typ := &module.TypeSection[module.FunctionSection[i]]

		codeSeg := &module.CodeSection[i]
		if codeSeg.GoFunc != nil {
			panic("TODO: host module")
		}

		fe.Init(wasm.Index(i), typ, codeSeg.LocalTypes, codeSeg.Body)

		// Lower Wasm to SSA.
		err := fe.LowerToSSA()
		if err != nil {
			return fmt.Errorf("wasm->ssa: %v", err)
		}

		// Run SSA-level optimization passes.
		ssaBuilder.Optimize()

		// Now our ssaBuilder contains the necessary information to further lower them to
		// machine code.
		body, err := be.Generate()
		if err != nil {
			return fmt.Errorf("ssa->machine code: %v", err)
		}

		totalSize += len(body)

		// TODO: optimize as zero copy.
		copied := make([]byte, len(body))
		copy(copied, body)
		bodies[i] = copied

		// Now we've generated machine code, so reset the backend's state,
		// make it ready for the next iteration.
		be.Reset()
	}

	if totalSize == 0 {
		// TODO: temporarily allowing empty code gen results until start implementing
		// backend.
		return nil
	}

	executable, err := platform.MmapCodeSegment(totalSize)
	if err != nil {
		panic(err)
	}
	cm.executable = executable
	cm.compiledFunctions = make([]compiledFunction, localFns)

	var offset int
	for i, b := range bodies {
		cm.compiledFunctions[i].offsetInExecutable = offset
		copy(executable[offset:], b)

		// Align 16-bytes boundary.
		offset = (offset + len(b) + 15) &^ 15
	}

	// TODO: handle relocations w.r.t direct function calls.

	if runtime.GOARCH == "arm64" {
		// On arm64, we cannot give all of rwx at the same time, so we change it to exec.
		if err = platform.MprotectRX(executable); err != nil {
			return err
		}
	}

	return nil
}

// Close implements wasm.Engine.
func (e *engine) Close() (err error) { panic("implement me") }

// CompiledModuleCount implements wasm.Engine.
func (e *engine) CompiledModuleCount() uint32 {
	e.mux.RLock()
	defer e.mux.RUnlock()
	return uint32(len(e.compiledModules))
}

// DeleteCompiledModule implements wasm.Engine.
func (e *engine) DeleteCompiledModule(m *wasm.Module) {
	e.mux.Lock()
	defer e.mux.Unlock()
	delete(e.compiledModules, m.ID)
}

// NewModuleEngine implements wasm.Engine.
func (e *engine) NewModuleEngine(*wasm.Module, *wasm.ModuleInstance) (wasm.ModuleEngine, error) {
	panic("implement me")
}

func (e *engine) addCompiledModule(m *wasm.Module, cm *compiledModule) {
	e.mux.Lock()
	defer e.mux.Unlock()
	e.compiledModules[m.ID] = cm
}
