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

package java

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwego/abcoder/lang/java/ipc"
	"github.com/cloudwego/abcoder/lang/java/pb"
)

const (
	MaxWaitDuration = 5 * time.Second

	// Java Parser JAR configuration
	javaParserVersion = "1.0.0"
	javaParserJarName = "java-parser.jar"

	// Legacy JDTLS configuration (deprecated)
	jdtlsVersion = "1.39.0-202408291433"
	jdtlsURL     = "https://download.eclipse.org/jdtls/milestones/1.39.0/jdt-language-server-1.39.0-202408291433.tar.gz"
)

// ParserConfig holds configuration for the Java Parser
type ParserConfig struct {
	// JarPath is the explicit path to the Java Parser JAR
	// If empty, will look for the JAR in standard locations
	JarPath string

	// JavaHome is the path to Java installation
	// If empty, uses system default
	JavaHome string

	// ResolveMavenDependencies enables Maven dependency resolution
	ResolveMavenDependencies bool

	// M2RepositoryPath is the path to Maven local repository
	// If empty, uses default ~/.m2/repository
	M2RepositoryPath string

	// ExtraJarPaths are additional JAR files to include in analysis
	ExtraJarPaths []string

	// IncludeExternalClasses includes external class information in results
	IncludeExternalClasses bool

	// Debug enables verbose logging
	Debug bool

	// Timeout for the entire analysis
	Timeout time.Duration
}

// DefaultParserConfig returns a default parser configuration
func DefaultParserConfig() *ParserConfig {
	jarPath := os.Getenv("JAVA_PARSER_JAR_PATH")
	if jarPath == "" {
		panic("JAVA_PARSER_JAR_PATH environment variable is required for Java Parser")
	}

	return &ParserConfig{
		ResolveMavenDependencies: true,
		IncludeExternalClasses:   false,
		Debug:                    false,
		JarPath:                  jarPath,
		Timeout:                  60 * time.Minute,
	}
}

func ParseRepositoryByIpc(ctx context.Context, repoPath string, config *ParserConfig) (*ipc.Converter, error) {
	if config == nil {
		config = DefaultParserConfig()
	}

	// Create IPC server configuration
	serverConfig := ipc.DefaultConfig()
	if config.JarPath != "" {
		serverConfig.JarPath = config.JarPath
	}
	serverConfig.JavaHome = config.JavaHome
	serverConfig.Debug = config.Debug

	if config.Timeout > 0 {
		serverConfig.ReadTimeout = config.Timeout
	}

	// Create analyzer config
	analyzerConfig := &pb.AnalyzerConfig{
		ResolveMavenDependencies: config.ResolveMavenDependencies,
		M2RepositoryPath:         config.M2RepositoryPath,
		ExtraJarPaths:            config.ExtraJarPaths,
		IncludeExternalClasses:   config.IncludeExternalClasses,
	}

	// Create server and start analysis
	server := ipc.NewJavaParserServer(serverConfig)
	defer server.Stop()

	responseChan, err := server.Start(ctx, repoPath, analyzerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to start Java Parser: %w", err)
	}

	// Convert responses to UniAST
	moduleName := filepath.Base(repoPath)
	converter := ipc.NewConverter(repoPath, moduleName)

	for resp := range responseChan {
		if err := converter.ProcessResponse(resp); err != nil {
			log.Printf("Warning: error processing response: %v", err)
		}
	}

	return converter, nil
}
