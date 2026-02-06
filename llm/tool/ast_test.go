/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tool

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

func TestASTTool_ToEinoTool(t *testing.T) {
	type fields struct {
		repo string
	}
	tests := []struct {
		name   string
		fields fields
		want   tool.BaseTool
	}{
		{
			name: "test",
			fields: fields{
				repo: "../../tmp/localsession.json",
			},
			want: utils.NewTool(
				&schema.ToolInfo{
					Name: "query_ast_node",
					Desc: "query the info of a AST node",
					ParamsOneOf: schema.NewParamsOneOfByParams(
						map[string]*schema.ParameterInfo{
							"id": {
								Type:     schema.Object,
								Desc:     "the id of the ast node",
								Required: true,
								SubParams: map[string]*schema.ParameterInfo{
									"build_package": {
										Type: schema.String,
										Desc: "the building build of the ast node belongs to, e.g. github.com/bytedance/sonic",
									},
									"version": {
										Type:     schema.String,
										Desc:     "the version of the building build, e.g. v1.0.0",
										Required: false,
									},
									"namespace": {
										Type: schema.String,
										Desc: "the namespace of the ast node belongs to, e.g. encoder/vm",
									},
									"name": {
										Type: schema.String,
										Desc: "the name of the ast node, e.g. Node.String",
									},
								},
							},
						},
					),
				},
				(&ASTReadTools{}).GetASTNode,
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewASTReadTools(ASTReadToolsOptions{
				// PatchOptions: patch.Options{
				// 	DefaultLanuage: uniast.Golang,
				// 	OutDir:         "./tmp",
				// 	RepoDir:        "../../tmp/localsession",
				// },
				RepoASTsDir: "../../testdata/asts",
			})
			for _, tool := range tr.tools {
				t.Logf("tool: %#v", tool)
			}
		})
	}
}

func TestASTTools_GetFileStructure(t *testing.T) {
	type fields struct {
		opts ASTReadToolsOptions
	}
	type args struct {
		in0 context.Context
		req GetFileStructReq
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *GetFileStructResp
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: "../../testdata/asts",
				},
			},
			args: args{
				in0: context.Background(),
				req: GetFileStructReq{
					RepoName: "localsession",
					FilePath: "backup/metainfo_test.go",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewASTReadTools(tt.fields.opts)
			got, err := tr.GetFileStructure(tt.args.in0, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ASTTools.GetFileStructure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ASTTools.GetFileStructure() = %v, want %v", got, tt.want)
			}
		})
	}
}

// var hertzRepo *uniast.Repository

// func TestMain(m *testing.M) {
// 	repox, err := uniast.LoadRepo("../../tmp/hertz.json")
// 	if err != nil {
// 		panic(err)
// 	}
// 	hertzRepo = repox
// 	os.Exit(m.Run())
// }

func TestASTTools_GetRepoStructure(t *testing.T) {

	type fields struct {
		opts ASTReadToolsOptions
	}
	type args struct {
		in0 context.Context
		req GetRepoStructReq
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *GetRepoStructResp
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: "../../testdata/asts",
				},
			},
			args: args{
				in0: context.Background(),
				req: GetRepoStructReq{
					RepoName:            "metainfo",
					ReturnPackageDetail: true,
				},
			},
			want: &GetRepoStructResp{
				Modules: []ModuleStruct{
					{
						ModPath: "metainfo",
						Packages: []PackageStruct{
							{PkgPath: "metainfo", Files: []FileStruct{{FilePath: "src/lib.rs"}}},
							{PkgPath: "metainfo::backward", Files: []FileStruct{{FilePath: "src/backward.rs"}}},
							{PkgPath: "metainfo::convert", Files: []FileStruct{{FilePath: "src/convert.rs"}}},
							{PkgPath: "metainfo::faststr_map", Files: []FileStruct{{FilePath: "src/faststr_map.rs"}}},
							{PkgPath: "metainfo::forward", Files: []FileStruct{{FilePath: "src/forward.rs"}}},
							{PkgPath: "metainfo::kv", Files: []FileStruct{{FilePath: "src/kv.rs"}}},
							{PkgPath: "metainfo::type_map", Files: []FileStruct{{FilePath: "src/type_map.rs"}}},
						},
					},
				},
			},
		},
		{
			name: "test",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: "../../testdata/asts",
				},
			},
			args: args{
				in0: context.Background(),
				req: GetRepoStructReq{
					RepoName:            "localsession",
					ReturnPackageDetail: true,
				},
			},
			want: &GetRepoStructResp{
				Modules: []ModuleStruct{
					{
						ModPath: "github.com/cloudwego/localsession",
						Packages: []PackageStruct{
							{
								PkgPath: "github.com/cloudwego/localsession",
								Files: []FileStruct{
									{FilePath: "gls.go"},
									{FilePath: "manager.go"},
									{FilePath: "session.go"},
									{FilePath: "stubs.go"},
								},
							},
							{
								PkgPath: "github.com/cloudwego/localsession [github.com/cloudwego/localsession.test]",
								Files: []FileStruct{
									{FilePath: "api_test.go"},
									{FilePath: "example_test.go"},
								},
							},
							{
								PkgPath: "github.com/cloudwego/localsession/backup",
								Files: []FileStruct{
									{FilePath: "backup/metainfo.go"},
								},
							},
							{
								PkgPath: "github.com/cloudwego/localsession/backup [github.com/cloudwego/localsession/backup.test]",
								Files: []FileStruct{
									{FilePath: "backup/metainfo_test.go"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "test_compress_suffix",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: "../../testdata/asts",
				},
			},
			args: args{
				in0: context.Background(),
				req: GetRepoStructReq{
					RepoName:            "metainfo",
					ReturnPackageDetail: false,
					CompressSuffix:      true,
				},
			},
			want: &GetRepoStructResp{
				Modules: []ModuleStruct{
					{
						ModPath: "metainfo",
						Packages: []PackageStruct{
							{PkgPath: "{{0}}"},
							{PkgPath: "{{0}}::backward"},
							{PkgPath: "{{0}}::convert"},
							{PkgPath: "{{0}}::faststr_map"},
							{PkgPath: "{{0}}::forward"},
							{PkgPath: "{{0}}::kv"},
							{PkgPath: "{{0}}::type_map"},
						},
					},
				},
				IsCompressed:   true,
				CompressVarMap: map[string]string{"0": "metainfo"},
			},
		},
		{
			name: "test_no_detail",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: "../../testdata/asts",
				},
			},
			args: args{
				in0: context.Background(),
				req: GetRepoStructReq{
					RepoName:            "metainfo",
					ReturnPackageDetail: false,
				},
			},
			want: &GetRepoStructResp{
				Modules: []ModuleStruct{
					{
						ModPath: "metainfo",
						Packages: []PackageStruct{
							{PkgPath: "metainfo"},
							{PkgPath: "metainfo::backward"},
							{PkgPath: "metainfo::convert"},
							{PkgPath: "metainfo::faststr_map"},
							{PkgPath: "metainfo::forward"},
							{PkgPath: "metainfo::kv"},
							{PkgPath: "metainfo::type_map"},
						},
					},
				},
			},
		},
	}
	sortResp := func(resp *GetRepoStructResp) {
		if resp == nil {
			return
		}
		sort.Slice(resp.Modules, func(i, j int) bool {
			return resp.Modules[i].ModPath < resp.Modules[j].ModPath
		})
		for i := range resp.Modules {
			mod := &resp.Modules[i]
			sort.Slice(mod.Packages, func(i, j int) bool {
				return mod.Packages[i].PkgPath < mod.Packages[j].PkgPath
			})
			for j := range mod.Packages {
				pkg := &mod.Packages[j]
				sort.Slice(pkg.Files, func(i, j int) bool {
					return pkg.Files[i].FilePath < pkg.Files[j].FilePath
				})
			}
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewASTReadTools(tt.fields.opts)
			got, err := tr.GetRepoStructure(tt.args.in0, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ASTTools.GetRepoStructure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			sortResp(got)
			sortResp(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ASTTools.GetRepoStructure() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestASTTools_GetPackageStructure(t *testing.T) {
	type fields struct {
		opts ASTReadToolsOptions
	}
	type args struct {
		ctx context.Context
		req GetPackageStructReq
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *GetPackageStructResp
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: "../../testdata/asts",
				},
			},
			args: args{
				ctx: context.Background(),
				req: GetPackageStructReq{
					RepoName: "localsession",
					ModPath:  "github.com/cloudwego/localsession",
					PkgPath:  "github.com/cloudwego/localsession/backup",
				},
			},
		},
		{
			name: "test",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: "../../testdata/asts",
				},
			},
			args: args{
				ctx: context.Background(),
				req: GetPackageStructReq{
					RepoName: "metainfo",
					ModPath:  "metainfo",
					PkgPath:  "metainfo::kv",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewASTReadTools(tt.fields.opts)
			got, err := tr.GetPackageStructure(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ASTTools.GetPackageStructure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ASTTools.GetPackageStructure() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestASTTools_GetASTNode(t *testing.T) {
	type fields struct {
		opts ASTReadToolsOptions
	}
	type args struct {
		in0    context.Context
		params GetASTNodeReq
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *GetASTNodeResp
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: "../../testdata/asts",
				},
			},
			args: args{
				in0: context.Background(),
				params: GetASTNodeReq{
					RepoName: "localsession",
					NodeIDs: []NodeID{
						{
							ModPath: "github.com/cloudwego/localsession",
							PkgPath: "github.com/cloudwego/localsession/backup",
							Name:    "RecoverCtxOnDemands",
						},
						{
							ModPath: "github.com/cloudwego/localsession",
							PkgPath: "github.com/cloudwego/localsession",
							Name:    "CurSession",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewASTReadTools(tt.fields.opts)
			got, err := tr.GetASTNode(tt.args.in0, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ASTTools.GetASTNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ASTTools.GetASTNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

// func TestASTTools_WriteASTNode(t *testing.T) {
// 	type fields struct {
// 		opts    ASTToolsOptions
// 		repo    *uniast.Repository
// 		patcher *patch.Patcher
// 		tools   map[string]tool.InvokableTool
// 	}
// 	type args struct {
// 		in0 context.Context
// 		req WriteASTNodeReq
// 	}
// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		args    args
// 		want    *WriteASTNodeResp
// 		wantErr bool
// 	}{
// 		{
// 			name: "add",
// 			fields: fields{
// 				opts: ASTToolsOptions{
// 					PatchOptions: patch.Options{
// 						DefaultLanuage: uniast.Golang,
// 						OutDir:         "../../tmp/hertz",
// 						RepoDir:        "../../tmp/hertz",
// 					},
// 				},
// 				repo: hertzRepo,
// 			},
// 			args: args{
// 				in0: context.Background(),
// 				req: WriteASTNodeReq{
// 					ID: uniast.Identity{
// 						ModPath: "github.com/cloudwego/hertz",
// 						PkgPath: "github.com/cloudwego/hertz/pkg/app",
// 						Name:    "RequestContext2",
// 					},
// 					Codes: `type RequestContext2 struct {
// 						RequestContext
// 					}`,
// 					File: "pkg/app/context.go",
// 					Type: "TYPE",
// 				},
// 			},
// 		},
// 		{
// 			name: "modify",
// 			fields: fields{
// 				opts: ASTToolsOptions{
// 					PatchOptions: patch.Options{
// 						DefaultLanuage: uniast.Golang,
// 						OutDir:         "../../tmp/hertz",
// 						RepoDir:        "../../tmp/hertz",
// 					},
// 				},
// 				repo: hertzRepo,
// 			},
// 			args: args{
// 				in0: context.Background(),
// 				req: WriteASTNodeReq{
// 					ID: uniast.Identity{
// 						ModPath: "github.com/cloudwego/hertz",
// 						PkgPath: "github.com/cloudwego/hertz",
// 						Name:    "Version",
// 					},
// 					Codes: `Version = "v2"`,
// 				},
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			tr := NewASTTools(tt.fields.repo, ASTToolsOptions{
// 				PatchOptions: tt.fields.opts.PatchOptions,
// 			})
// 			got, err := tr.WriteASTNode(tt.args.in0, tt.args.req)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("ASTTools.WriteASTNode() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			_ = got
// 			// if !reflect.DeepEqual(got, tt.want) {
// 			// 	t.Errorf("ASTTools.WriteASTNode() = %v, want %v", got, tt.want)
// 			// }
// 		})
// 	}
// }

func TestCompressSuffix(t *testing.T) {
	tests := []struct {
		name         string
		index        int
		packages     []string // Input package paths
		wantPrefix   string
		wantPackages []string // Expected package paths after compression
	}{
		{
			name:         "empty packages",
			index:        1,
			packages:     []string{},
			wantPrefix:   "",
			wantPackages: []string{},
		},
		{
			name:         "no common prefix",
			index:        1,
			packages:     []string{"a/b", "c/d"},
			wantPrefix:   "",
			wantPackages: []string{"a/b", "c/d"},
		},
		{
			name:         "single package",
			index:        1,
			packages:     []string{"github.com/cloudwego/abcoder"},
			wantPrefix:   "github.com/cloudwego/abcoder",
			wantPackages: []string{"{{1}}"},
		},
		{
			name:         "common prefix all match",
			index:        2,
			packages:     []string{"github.com/cloudwego/abcoder/a", "github.com/cloudwego/abcoder/b"},
			wantPrefix:   "github.com/cloudwego/abcoder/",
			wantPackages: []string{"{{2}}a", "{{2}}b"},
		},
		{
			name:  "common prefix partial match (limit check)",
			index: 3,
			packages: []string{
				"prefix/1", "prefix/2", "prefix/3", "prefix/4", "prefix/5",
				"prefix/6", "prefix/7", "prefix/8", "prefix/9", "prefix/10",
				"other/11", // 11th element, should not affect prefix calculation of first 10
			},
			// First 10 are "prefix/...", common prefix is "prefix/"
			wantPrefix: "prefix/",
			wantPackages: []string{
				"{{3}}1", "{{3}}2", "{{3}}3", "{{3}}4", "{{3}}5",
				"{{3}}6", "{{3}}7", "{{3}}8", "{{3}}9", "{{3}}10",
				"other/11", // Should not be replaced because it doesn't match "prefix/"
			},
		},
		{
			name:  "common prefix affects all",
			index: 4,
			packages: []string{
				"long/path/1", "long/path/2", "long/path/3", "long/path/4", "long/path/5",
				"long/path/6", "long/path/7", "long/path/8", "long/path/9", "long/path/10",
				"long/path/11", // 11th element, should be replaced if it matches
			},
			wantPrefix: "long/path/",
			wantPackages: []string{
				"{{4}}1", "{{4}}2", "{{4}}3", "{{4}}4", "{{4}}5",
				"{{4}}6", "{{4}}7", "{{4}}8", "{{4}}9", "{{4}}10",
				"{{4}}11",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mod := ModuleStruct{}
			for _, p := range tt.packages {
				mod.Packages = append(mod.Packages, PackageStruct{PkgPath: uniast.PkgPath(p)})
			}

			gotPrefix := compressSuffix(tt.index, mod)

			if gotPrefix != tt.wantPrefix {
				t.Errorf("compressSuffix() prefix = %v, want %v", gotPrefix, tt.wantPrefix)
			}

			gotPackages := []string{}
			for _, p := range mod.Packages {
				gotPackages = append(gotPackages, string(p.PkgPath))
			}

			if !reflect.DeepEqual(gotPackages, tt.wantPackages) {
				t.Errorf("compressSuffix() packages = %v, want %v", gotPackages, tt.wantPackages)
			}
		})
	}
}
