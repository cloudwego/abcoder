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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPythonSpecWorkSpaceAllowsNestedPyprojectFiles(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "pyproject.toml"), nil, 0o600))
	nested := filepath.Join(root, "vendor", "pylsp")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "pyproject.toml"), nil, 0o600))

	modules, err := (&PythonSpec{}).WorkSpace(root)
	require.NoError(t, err)
	absRoot, err := filepath.Abs(root)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"current": absRoot}, modules)
}

func TestPythonSpecWorkSpaceRejectsMissingRoot(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	modules, err := (&PythonSpec{}).WorkSpace(missing)
	require.Error(t, err)
	require.Nil(t, modules)
}
