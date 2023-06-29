package arm64

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

// Test_regNames tests whether regNames is initialized correctly.
func Test_regNames(t *testing.T) {
	for i := x0; i < numRegisters; i++ {
		require.NotEqual(t, "", regNames[i])
	}
}
