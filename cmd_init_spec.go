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

package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/abcoder/llm/log"
)

//go:embed docs/.claude
var claudeFS embed.FS

// claudeConfig represents the Claude Code configuration structure
type claudeConfig struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
}

type mcpServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// runInitSpec implements the init-spec command
func runInitSpec(targetDir string) error {
	if targetDir == "" {
		// Default to current directory if not specified
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		targetDir = cwd
	}

	// Ensure targetDir is absolute
	targetDirAbs, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 1. Copy docs/.claude to targetDir/.claude
	claudeDestDir := filepath.Join(targetDirAbs, ".claude")
	if err := copyEmbeddedDir("docs/.claude", claudeDestDir, targetDirAbs); err != nil {
		return fmt.Errorf("failed to copy .claude directory: %w", err)
	}
	log.Info("Copied .claude directory to %s", claudeDestDir)

	// 2. Get home directory for MCP server configuration
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// 3. Configure MCP servers in ~/.claude.json
	// Get asts directory path from parse.sh hook (default ~/.asts)
	astsDir := filepath.Join(homeDir, ".asts")

	// Create asts directory if it doesn't exist
	if err := os.MkdirAll(astsDir, 0755); err != nil {
		return fmt.Errorf("failed to create asts directory: %w", err)
	}

	claudeConfigPath := filepath.Join(homeDir, ".claude.json")
	if err := configureMCPServers(claudeConfigPath, astsDir); err != nil {
		return fmt.Errorf("failed to configure MCP servers: %w", err)
	}
	log.Info("Configured MCP servers in %s", claudeConfigPath)

	// 4. Print success message
	printSuccessMessage(targetDirAbs, claudeConfigPath, astsDir)

	return nil
}

// copyEmbeddedDir copies an embedded directory to a destination directory
func copyEmbeddedDir(srcPath string, destDir string, projectRootDir string) error {
	// First, ensure the destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
	}

	// Track md files to process after copying
	var mdFilesToReplace []string

	err := fs.WalkDir(claudeFS, srcPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from srcPath
		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			// Create directory
			return os.MkdirAll(destPath, 0755)
		}

		// Ensure parent directory exists before writing file
		parentDir := filepath.Dir(destPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
		}

		// Rename command files with abcoder: prefix
		if strings.HasPrefix(relPath, "commands/") {
			baseName := filepath.Base(relPath)
			switch baseName {
			case "recheck.md":
				destPath = filepath.Join(filepath.Dir(destPath), "abcoder:recheck.md")
			case "schedule.md":
				destPath = filepath.Join(filepath.Dir(destPath), "abcoder:schedule.md")
			case "task.md":
				destPath = filepath.Join(filepath.Dir(destPath), "abcoder:task.md")
			}
		}

		// Copy file
		data, err := claudeFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", destPath, err)
		}

		// Set executable permission for shell scripts
		if strings.HasSuffix(relPath, ".sh") {
			if err := os.Chmod(destPath, 0755); err != nil {
				return fmt.Errorf("failed to set executable permission for %s: %w", destPath, err)
			}
		}

		// Track md and json files for placeholder replacement
		if strings.HasSuffix(relPath, ".md") || strings.HasSuffix(relPath, ".json") || strings.HasSuffix(relPath, "prompt.sh") {
			mdFilesToReplace = append(mdFilesToReplace, destPath)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Replace {{CLAUDE_HOME_PATH}} placeholder in md files with project root directory
	for _, mdFile := range mdFilesToReplace {
		if err := replaceClaudeHomePlaceholder(mdFile, projectRootDir); err != nil {
			log.Info("Failed to replace placeholder in %s: %v", mdFile, err)
		}
	}

	return nil
}

// replaceClaudeHomePlaceholder replaces {{CLAUDE_HOME_PATH}} with actual project root directory path
func replaceClaudeHomePlaceholder(filePath string, projectRootDir string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	content := string(data)
	newContent := strings.ReplaceAll(content, "{{CLAUDE_HOME_PATH}}", projectRootDir)

	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

// configureMCPServers configures MCP servers in the Claude config file
func configureMCPServers(configPath string, astsDir string) error {
	var config claudeConfig

	// Read existing config if it exists
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Initialize mcpServers map if nil
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]mcpServerConfig)
	}

	// Add/Update abcoder MCP server
	config.MCPServers["abcoder"] = mcpServerConfig{
		Command: "abcoder",
		Args:    []string{"mcp", astsDir},
	}

	// Add sequential-thinking MCP server
	config.MCPServers["sequential-thinking"] = mcpServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-sequential-thinking"},
	}

	// Write the config file
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// printSuccessMessage prints a success message with instructions
func printSuccessMessage(targetDir string, configPath string, astsDir string) {
	fmt.Printf(`
✓ ABCoder Claude Code integration setup completed!

Configuration files:
  .claude directory:     %s
  Claude Code config:   %s
  AST storage directory: %s

MCP servers configured:
  - abcoder: for code analysis using AST
  - sequential-thinking: for complex problem decomposition

Next steps:
  1. Ensure abcoder is installed and in your PATH:
     go install github.com/cloudwego/abcoder@latest

  2. Restart Claude Code to apply the configuration

  3. Use ABCoder tools in Claude Code:
     - /abcoder:schedule - Analyze codebase and design solution
     - /abcoder:task - Create coding task
     - /abcoder:recheck - Verify solution

For more information, see:
  - https://github.com/cloudwego/abcoder
`, targetDir, configPath, astsDir)
}
