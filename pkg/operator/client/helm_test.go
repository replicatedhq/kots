package client

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_stopHelmForwardsSIGTERM(t *testing.T) {
	t.Cleanup(func() {
		helmBinary = "helm"
		helmCtx, cancelHelmCtx = context.WithCancel(context.Background())
		helmRunning = sync.RWMutex{}
	})

	// use a shell in place of helm that exits with 143 on SIGTERM. a SIGKILL would not run the trap.
	helmBinary = "sh"
	readyFile := filepath.Join(t.TempDir(), "ready")
	exitedFile := filepath.Join(t.TempDir(), "exited")
	// the delay before exiting checks that stopHelm waits for helm to exit
	script := `trap 'kill $!; sleep 1; touch "$1"; exit 143' TERM; sleep 60 >/dev/null 2>&1 & touch "$0"; wait`

	errCh := make(chan error, 1)
	go func() {
		_, _, err := runHelm("-c", script, readyFile, exitedFile)
		errCh <- err
	}()

	require.Eventually(t, func() bool {
		_, err := os.Stat(readyFile)
		return err == nil
	}, 10*time.Second, 10*time.Millisecond)

	start := time.Now()
	require.True(t, stopHelm(10*time.Second))
	_, err := os.Stat(exitedFile)
	require.NoError(t, err)
	require.Less(t, time.Since(start), 5*time.Second)

	err = <-errCh
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, 143, exitErr.ExitCode(), "expected helm to exit from SIGTERM, got %v", err)

	// in production the held write lock blocks new helm runs. this only checks that the cancelled
	// context also stops helm from starting.
	helmRunning.Unlock()
	_, _, err = runHelm("-c", "exit 0")
	require.ErrorIs(t, err, context.Canceled)
}
