// Copyright 2025 CloudWeGo Authors
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

package writer

import (
	"strings"
	"testing"

	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter_SplitImportsAndCodes(t *testing.T) {
	w := NewWriter(Options{})

	tests := []struct {
		name        string
		src         string
		wantCodes   string
		wantImports []uniast.Import
	}{
		{
			name: "full source with package and imports",
			src: `package com.example;

import java.util.List;
import java.util.Map;

public class Foo {
}`,
			wantCodes: `public class Foo {
}`,
			wantImports: []uniast.Import{
				{Path: "java.util.List"},
				{Path: "java.util.Map"},
			},
		},
		{
			name:        "no imports",
			src:         `public class Bar {}`,
			wantCodes:   `public class Bar {}`,
			wantImports: nil,
		},
		{
			name: "static import",
			src: `import static org.junit.Assert.assertEquals;

public class Test {}`,
			wantCodes: `public class Test {}`,
			wantImports: func() []uniast.Import {
				alias := "static"
				return []uniast.Import{
					{Path: "org.junit.Assert.assertEquals", Alias: &alias},
				}
			}(),
		},
		{
			name: "only package line",
			src: `package com.example;

public class Baz {}`,
			wantCodes:   `public class Baz {}`,
			wantImports: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes, imports, err := w.SplitImportsAndCodes(tt.src)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCodes, codes)
			assert.Equal(t, tt.wantImports, imports)
		})
	}
}

func TestWriter_IdToImport(t *testing.T) {
	w := NewWriter(Options{})

	imp, err := w.IdToImport(uniast.Identity{
		PkgPath: "com.example.service",
		Name:    "UserService",
	})
	require.NoError(t, err)
	assert.Equal(t, "com.example.service.UserService", imp.Path)
}

func TestWriter_PatchImports(t *testing.T) {
	w := NewWriter(Options{})

	tests := []struct {
		name    string
		file    string
		impts   []uniast.Import
		want    string
		wantErr bool
	}{
		{
			name: "add import to file with existing imports",
			file: `package com.example;

import java.util.List;

public class Foo {}`,
			impts: []uniast.Import{
				{Path: "java.util.Map"},
			},
			want: `package com.example;

import java.util.List;
import java.util.Map;

public class Foo {}`,
		},
		{
			name: "no new imports needed (dedup)",
			file: `package com.example;

import java.util.List;

public class Foo {}`,
			impts: []uniast.Import{
				{Path: "java.util.List"},
			},
			want: `package com.example;

import java.util.List;

public class Foo {}`,
		},
		{
			name: "add import to file without imports",
			file: `package com.example;

public class Bar {}`,
			impts: []uniast.Import{
				{Path: "java.util.List"},
			},
			want: `package com.example;

import java.util.List;

public class Bar {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := w.PatchImports(tt.impts, []byte(tt.file))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestWriter_CreateFile(t *testing.T) {
	w := NewWriter(Options{})

	fi := &uniast.File{
		Package: "com.example.service",
		Imports: []uniast.Import{
			{Path: "java.util.List"},
			{Path: "java.util.Map"},
		},
	}
	mod := &uniast.Module{
		Name: "com.example:myapp:1.0.0",
	}

	got, err := w.CreateFile(fi, mod)
	require.NoError(t, err)

	want := "package com.example.service;\n\nimport java.util.List;\nimport java.util.Map;\n\n"
	assert.Equal(t, want, string(got))
}

func TestWriter_CreateFile_EmptyPackage(t *testing.T) {
	w := NewWriter(Options{})

	fi := &uniast.File{}
	mod := &uniast.Module{}

	_, err := w.CreateFile(fi, mod)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "package name is empty")
}

func TestMergeImports(t *testing.T) {
	priors := []uniast.Import{
		{Path: "java.util.List"},
		{Path: "java.util.Map"},
	}
	subs := []uniast.Import{
		{Path: "java.util.Map"},
		{Path: "java.io.File"},
	}
	merged := mergeImports(priors, subs)
	assert.Len(t, merged, 3)
	assert.Equal(t, "java.util.List", merged[0].Path)
	assert.Equal(t, "java.util.Map", merged[1].Path)
	assert.Equal(t, "java.io.File", merged[2].Path)
}

func TestWriteImport(t *testing.T) {
	var sb strings.Builder
	impts := []uniast.Import{
		{Path: "java.util.List"},
		{Path: "java.util.Map"},
	}
	writeImport(&sb, impts)
	want := "import java.util.List;\nimport java.util.Map;\n\n"
	assert.Equal(t, want, sb.String())
}

func TestWriteSingleImport_Static(t *testing.T) {
	var sb strings.Builder
	alias := "static"
	imp := uniast.Import{Path: "org.junit.Assert.assertEquals", Alias: &alias}
	writeSingleImport(&sb, imp)
	assert.Equal(t, "import static org.junit.Assert.assertEquals;\n", sb.String())
}
