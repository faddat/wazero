package arm64

import (
	"fmt"
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

// Test_regNames tests whether regNames is initialized correctly.
func Test_regNames(t *testing.T) {
	for i := w1; i < numRegisters; i++ {
		fmt.Println(i)
		require.NotEqual(t, "", regNames[i])
	}
}
