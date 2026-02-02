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

package ipc

import (
	"path/filepath"
	"strings"

	"github.com/cloudwego/abcoder/lang/java/pb"
	"github.com/cloudwego/abcoder/lang/uniast"
)

type Converter struct {
	repo       *uniast.Repository
	repoPath   string
	moduleName string

	// Caches
	fileCache   map[string]*uniast.File
	moduleCache map[string]*uniast.Module

	JdkClassCache       map[string]*pb.ClassInfo
	LocalClassCache     map[string]*pb.ClassInfo
	UnknowClassCache    map[string]*pb.ClassInfo
	ThirdPartClassCache map[string]*pb.ClassInfo
}

// NewConverter creates a new converter for the given repository
func NewConverter(repoPath string, moduleName string) *Converter {
	repo := uniast.NewRepository(moduleName)
	repo.Path = repoPath

	c := &Converter{
		repo:                &repo,
		repoPath:            repoPath,
		moduleName:          moduleName,
		fileCache:           make(map[string]*uniast.File),
		moduleCache:         make(map[string]*uniast.Module),
		JdkClassCache:       make(map[string]*pb.ClassInfo),
		LocalClassCache:     make(map[string]*pb.ClassInfo),
		UnknowClassCache:    make(map[string]*pb.ClassInfo),
		ThirdPartClassCache: make(map[string]*pb.ClassInfo),
	}
	// 确保默认 module 存在（即使只收到 progress/summary）
	c.getOrCreateModule(moduleName)
	return c
}

// ConvertResponses 将 Java Parser 的流式响应列表转换为 UniAST Repository。
func ConvertResponses(repoPath string, moduleName string, responses []*pb.AnalyzeResponse) (*uniast.Repository, error) {
	conv := NewConverter(repoPath, moduleName)
	for _, resp := range responses {
		if err := conv.ProcessResponse(resp); err != nil {
			return conv.Repository(), err
		}
	}
	return conv.Repository(), nil
}

// Repository returns the converted UniAST repository
func (c *Converter) Repository() *uniast.Repository {
	return c.repo
}

// ProcessResponse processes a single AnalyzeResponse and updates the repository
func (c *Converter) ProcessResponse(resp *pb.AnalyzeResponse) error {
	if resp == nil {
		return nil
	}

	switch resp.PayloadType {
	case pb.PAYLOAD_FILE_INFO:
		return c.processFileInfo(resp.GetFileInfo())
	case pb.PAYLOAD_CLASS_INFO:
		return c.processClassInfo(resp.GetClassInfo())
	case pb.PAYLOAD_METHOD_CALL:
		return nil
	case pb.PAYLOAD_PROGRESS:
		// 进度不影响仓库结构
		return nil
	case pb.PAYLOAD_SUMMARY:
		// 汇总不影响仓库结构
		return nil
	case pb.PAYLOAD_ERROR:
		// 错误不影响仓库结构（由调用方处理）
		return nil
	}
	return nil
}

// processFileInfo converts FileInfo to UniAST File
func (c *Converter) processFileInfo(info *pb.FileInfo) error {
	if info == nil {
		return nil
	}

	// Get or create file
	file := c.getOrCreateFile(info.FilePath)

	// Extract package from file path
	file.Package = extractPackageFromPath(info.FilePath)

	return nil
}

// processClassInfo converts ClassInfo to UniAST Type and Functions
func (c *Converter) processClassInfo(info *pb.ClassInfo) error {
	if info == nil {
		return nil
	}

	err, _ := putCache(info, c)
	if err != nil {
		return err
	}
	for _, dep := range info.Dependencies {
		if dep.SourceType == pb.SourceType_SOURCE_TYPE_JDK && dep.ClassName != "" {
			if _, ok := c.JdkClassCache[dep.ClassName]; !ok {
				depPoint := &pb.ClassInfo{
					ClassName: dep.ClassName,
					Source: &pb.SourceInfo{
						Type: pb.SourceType_SOURCE_TYPE_JDK,
					},
				}
				putCache(depPoint, c)
			}
		}
		if dep.SourceType == pb.SourceType_SOURCE_TYPE_UNKNOWN && dep.ClassName != "" {
			if _, ok := c.UnknowClassCache[dep.ClassName]; !ok {
				depPoint := &pb.ClassInfo{
					ClassName: dep.ClassName,
					Source: &pb.SourceInfo{
						Type: pb.SourceType_SOURCE_TYPE_UNKNOWN,
					},
				}
				putCache(depPoint, c)
			}
		}
		if (dep.SourceType == pb.SourceType_SOURCE_TYPE_MAVEN || dep.SourceType == pb.SourceType_SOURCE_TYPE_EXTERNAL_JAR) && dep.ClassName != "" {
			if _, ok := c.ThirdPartClassCache[dep.ClassName]; !ok {
				depPoint := &pb.ClassInfo{
					ClassName: dep.ClassName,
					Source: &pb.SourceInfo{
						Type: pb.SourceType_SOURCE_TYPE_MAVEN,
					},
				}
				putCache(depPoint, c)
			}
		}
	}

	return nil
}

func putCache(info *pb.ClassInfo, c *Converter) (error, bool) {
	// Cache class info
	switch info.Source.Type {
	case pb.SourceType_SOURCE_TYPE_JDK:
		c.JdkClassCache[info.ClassName] = info
	case pb.SourceType_SOURCE_TYPE_LOCAL:
		c.LocalClassCache[info.ClassName] = info
	case pb.SourceType_SOURCE_TYPE_UNKNOWN:
		c.UnknowClassCache[info.ClassName] = info
	case pb.SourceType_SOURCE_TYPE_MAVEN, pb.SourceType_SOURCE_TYPE_EXTERNAL_JAR:
		c.ThirdPartClassCache[info.ClassName] = info
	default:
		return nil, true
	}
	return nil, false
}

// getOrCreateModule gets or creates a module
func (c *Converter) getOrCreateModule(name string) *uniast.Module {
	if mod, ok := c.moduleCache[name]; ok {
		return mod
	}

	mod := uniast.NewModule(name, "", uniast.Java)
	c.moduleCache[name] = mod
	c.repo.SetModule(name, mod)
	return mod
}

// getOrCreateFile gets or creates a file
func (c *Converter) getOrCreateFile(filePath string) *uniast.File {
	if file, ok := c.fileCache[filePath]; ok {
		return file
	}

	file := uniast.NewFile(filePath)
	c.fileCache[filePath] = file

	// Add file to appropriate module
	mod := c.getOrCreateModule(c.moduleName)
	mod.Files[filePath] = file

	return file
}

// extractPackageFromPath extracts package name from file path
func extractPackageFromPath(filePath string) string {
	// Remove file extension and convert path separators to dots
	dir := filepath.Dir(filePath)

	// Try to find src/main/java or src/ prefix
	if idx := strings.Index(dir, "src/main/java/"); idx != -1 {
		return strings.ReplaceAll(dir[idx+len("src/main/java/"):], "/", ".")
	}
	if idx := strings.Index(dir, "src/"); idx != -1 {
		return strings.ReplaceAll(dir[idx+len("src/"):], "/", ".")
	}

	return strings.ReplaceAll(dir, "/", ".")
}
