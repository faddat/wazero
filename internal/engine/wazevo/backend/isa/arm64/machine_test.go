package arm64

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestMachine_insertAtHead(t *testing.T) {
	t.Run("no head", func(t *testing.T) {
		m := &machine{}
		i := &instruction{kind: condBr}
		m.insertAtHead(i)
		require.Equal(t, i, m.head)
		require.Equal(t, i, m.tail)
	})
	t.Run("has head", func(t *testing.T) {
		prevHead := &instruction{kind: br}
		m := &machine{head: prevHead, tail: prevHead}
		i := &instruction{kind: condBr}
		m.insertAtHead(i)
		require.Equal(t, i, m.head)
		require.Equal(t, prevHead, m.tail)
		require.Equal(t, nil, prevHead.next)
		require.Equal(t, i, prevHead.prev)
		require.Equal(t, prevHead, i.next)
		require.Equal(t, nil, i.prev)
	})
}
