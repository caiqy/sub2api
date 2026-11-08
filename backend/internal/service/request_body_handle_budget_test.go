package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/stretchr/testify/require"
)

func TestRequestBodyHandleBudgetSharedAcrossMaterializations(t *testing.T) {
	for _, threshold := range []int64{1, 1024} {
		t.Run(string(rune('a'+threshold%2)), func(t *testing.T) {
			budget := httputil.NewBodyMemoryBudget(128)
			ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
			defer release()
			handle, err := NewRequestBodyHandleFromBytes(bytes.Repeat([]byte("x"), 64), RequestBodyHandleOptions{Context: ctx, SpoolThresholdBytes: threshold, TempDir: t.TempDir()})
			require.NoError(t, err)
			defer CleanupRequestBodyHandle(handle)
			_, err = handle.ReadAll()
			require.NoError(t, err)
			if threshold == 1 {
				_, err = handle.ReadAll()
				require.NoError(t, err)
			}
			_, err = handle.ReadAll()
			require.ErrorIs(t, err, httputil.ErrRequestBodyMemoryBudget)
			release()
			other, done := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
			defer done()
			require.NoError(t, httputil.ReserveRequestBodyMemory(other, 128))
		})
	}
}
