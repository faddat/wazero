package arm64

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func Test_asImm12(t *testing.T) {
	v, shift, ok := asImm12(0xfff)
	require.True(t, ok)
	require.True(t, shift == 0)
	require.Equal(t, uint16(0xfff), v)

	_, _, ok = asImm12(0xfff << 1)
	require.False(t, ok)

	v, shift, ok = asImm12(0xabc << 12)
	require.True(t, ok)
	require.True(t, shift == 1)
	require.Equal(t, uint16(0xabc), v)

	_, _, ok = asImm12(0xabc<<12 | 0b1)
	require.False(t, ok)
}
