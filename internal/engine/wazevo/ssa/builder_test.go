package ssa

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestNewBuilder(t *testing.T) {
	b := NewBuilder()
	require.NotNil(t, b)
}
