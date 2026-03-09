package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileMemory(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.pprof")

	err := profileMemory(tmpFile)
	require.NoError(t, err)

	info, err := os.Stat(tmpFile)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "profile file should not be empty")
}

func TestProfileMemory_InvalidPath(t *testing.T) {
	err := profileMemory("/nonexistent/directory/test.pprof")
	assert.Error(t, err)
}

func TestWorkload(t *testing.T) {
	// workload только выполняет бизнес-операции — убеждаемся, что не паникует
	assert.NotPanics(t, func() {
		workload()
	})
}

func TestRun_NoArgs(t *testing.T) {
	orig := os.Args
	os.Args = []string{"profiler"}
	t.Cleanup(func() { os.Args = orig })

	err := run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "использование")
}

func TestRun_WithValidArg(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "out.pprof")
	orig := os.Args
	os.Args = []string{"profiler", tmpFile}
	t.Cleanup(func() { os.Args = orig })

	err := run()
	assert.NoError(t, err)
}
