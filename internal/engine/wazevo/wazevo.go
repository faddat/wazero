package wazevo

import (
	"context"
	"sync"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/internal/filecache"
	"github.com/tetratelabs/wazero/internal/wasm"
)

type (
	// engine implements wasm.Engine.
	engine struct {
		compiledModules map[wasm.ModuleID]*compiledModule
		mux             sync.RWMutex
	}

	// compiledModule is a compiled variant of a wasm.Module and ready to be used for instantiation.
	compiledModule struct{}
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

	panic("implement me")
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
