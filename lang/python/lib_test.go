// Copyright 2026 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package python

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPylspInstallPathUsesUserCache(t *testing.T) {
	cacheDir, err := os.UserCacheDir()
	require.NoError(t, err)

	got, err := pylspInstallPath()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(cacheDir, "abcoder", "pylsp"), got)
	require.True(t, filepath.IsAbs(got))
}

func TestEnsurePylspSourceReusesCachedCheckout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "abcoder", "pylsp")
	require.NoError(t, os.MkdirAll(filepath.Join(path, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(path, "pyproject.toml"), nil, 0o600))

	require.NoError(t, ensurePylspSource(path))
}

func TestEnsurePylspSourceRejectsInvalidCacheDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "abcoder", "pylsp")
	require.NoError(t, os.MkdirAll(path, 0o755))

	err := ensurePylspSource(path)
	require.ErrorContains(t, err, "not a git checkout")
}

func TestEnsurePylspSourcePublishesConcurrentCloneAtomically(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "abcoder")
	path := filepath.Join(parent, "pylsp")
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	clone := func(tempPath string) error {
		if err := os.MkdirAll(filepath.Join(tempPath, ".git"), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(tempPath, "pyproject.toml"), nil, 0o600); err != nil {
			return err
		}
		ready <- struct{}{}
		<-release
		return nil
	}

	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- ensurePylspSourceWithClone(path, clone)
		}()
	}
	<-ready
	<-ready
	close(release)
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	require.NoError(t, validatePylspSource(path))
	entries, err := os.ReadDir(parent)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "pylsp", entries[0].Name())
}

func TestEnsurePylspSourceCleansUpInterruptedClone(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "abcoder")
	path := filepath.Join(parent, "pylsp")
	cloneErr := errors.New("clone interrupted")

	err := ensurePylspSourceWithClone(path, func(tempPath string) error {
		require.NoError(t, os.MkdirAll(filepath.Join(tempPath, ".git"), 0o755))
		return cloneErr
	})
	require.ErrorIs(t, err, cloneErr)
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
	entries, err := os.ReadDir(parent)
	require.NoError(t, err)
	require.Empty(t, entries)
}
