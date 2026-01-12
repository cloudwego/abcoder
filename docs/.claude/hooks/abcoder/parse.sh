#!/bin/bash

# 修正并简化的 get_basename 函数
get_basename() {
  # 检查输入是否为 "." 或 ".."
  if [[ "$1" == "." || "$1" == ".." ]]; then
    # 对于当前或上级目录，使用 basename 结合 pwd
    basename "$(cd "$1" && pwd)"
  else
    # 对于其他路径，直接使用 basename
    basename "$1"
  fi
}

# 新增：检测项目语言并获取仓库标识
detect_project_info() {
  local target_dir="$1"
  local project_info=""

  # 1. 优先检测 Go 项目（判断 go.mod 或 main.go）
  if [[ -f "${target_dir}/go.mod" ]]; then
    # 从 go.mod 中提取 module name
    local module_name=$(grep "^module " "${target_dir}/go.mod" | head -1 | awk '{print $2}')
    if [[ -n "$module_name" ]]; then
      echo "go|${module_name}"
      return 0
    fi
    # 如果无法获取 module name，使用 main.go 所在目录
    if [[ -f "${target_dir}/main.go" ]]; then
      echo "go|$(get_basename "$target_dir")"
      return 0
    fi
  fi

  # 2. 检测 TypeScript 项目（判断 package.json 或 tsconfig.json）
  if [[ -f "${target_dir}/package.json" ]]; then
    # 从 package.json 中提取 name
    local package_name=$(jq -r '.name // empty' "${target_dir}/package.json" 2>/dev/null)
    if [[ -n "$package_name" && "$package_name" != "null" ]]; then
      echo "ts|${package_name}"
      return 0
    fi
  fi

  # 3. 检测 TypeScript 项目（判断 tsconfig.json 或 .ts/.tsx 文件）
  if [[ -f "${target_dir}/tsconfig.json" ]]; then
    echo "ts|$(get_basename "$target_dir")"
    return 0
  fi

  # 统计 .ts 和 .tsx 文件数量（排除 node_modules 目录）
  local ts_file_count=$(find "${target_dir}" -type f -not -path "*/node_modules/*" \( -name "*.ts" -o -name "*.tsx" \) | wc -l)
  if [[ $ts_file_count -gt 0 ]]; then
    echo "ts|$(get_basename "$target_dir")"
    return 0
  fi

  # 4. 未检测到目标语言
  echo "unknown|$(get_basename "$target_dir")"
  return 1
}

# 直接映射 abc 为别名，处理 parse <语言> <仓库路径> 形式的参数
abc() {
  # 检查命令格式是否为 "parse <语言> <仓库路径>"
  if [ $# -eq 3 ] && [ "$1" = "parse" ]; then
    local lang="$2"
    local repo_path="$3"
    local repo_name=$(get_basename "$repo_path")

    # 确保输出目录存在
    mkdir -p ~/.asts/

    # 执行实际命令
    abcoder parse "${lang}" "${repo_path}" -o "~/.asts/${repo_name}.json"
  else
    # 如果不是预期的 parse 命令格式，直接将参数传递给原始 abcoder 命令
    abcoder "$@"
  fi
}

# LOG_FILE="/tmp/claude-hook-debug.log"

input=$(cat)
repo_name=$(echo "$input" | jq -r '.tool_input.repo_name // ""')
cwd=$(echo "$input" | jq -r '.cwd // ""')

# echo "=== $(date) ===" >> "$LOG_FILE"
# echo "repo_name: $repo_name" >> "$LOG_FILE"
# echo "cwd: $cwd" >> "$LOG_FILE"

# 复用现有的 get_basename 函数
current_base_name=$(get_basename "$cwd")

# 检测项目信息（语言和仓库标识）
project_info=$(detect_project_info "$cwd")
project_lang=$(echo "$project_info" | cut -d'|' -f1)
project_identifier=$(echo "$project_info" | cut -d'|' -f2)

# echo "Detected project language: $project_lang" >> "$LOG_FILE"
# echo "Detected project identifier: $project_identifier" >> "$LOG_FILE"

if [ "$repo_name" = "$cwd" ] || [ "$current_base_name" = "$repo_name" ] || [ "$project_identifier" = "$repo_name" ]; then
  # echo "Path or identifier matched, executing abc parse..." >> "$LOG_FILE"

  # 检查是否检测到目标语言
  if [[ "$project_lang" == "unknown" ]]; then
    jq -n '{
      "decision": "block",
      "reason": "未检测到支持的语言（仅支持 Go 和 TypeScript）",
      "hookSpecificOutput": {
        "hookEventName": "PreToolUse",
        "additionalContext": "请确保项目是 Go 或 TypeScript 类型"
      }
    }'
    exit 0
  fi

  # 捕获标准输出和错误输出
  output_file=$(mktemp)
  error_file=$(mktemp)

  # 修改：使用检测到的语言执行 parse 命令（替换原有的固定 ts）
  if abcoder parse "$project_lang" . >"$output_file" 2>"$error_file"; then
    # echo "abc parse succeeded" >> "$LOG_FILE"
    # cat "$output_file" >> "$LOG_FILE"

    jq -n --arg lang "$project_lang" '{
          "systemMessage": "abcoder parse 已成功完成（语言：\($lang)）。AST文件已生成，可以继续分析代码。"
        }'
  else
    exit_code=$?
    # echo "abc parse FAILED with exit code $exit_code" >> "$LOG_FILE"
    # echo "STDOUT:" >> "$LOG_FILE"
    # cat "$output_file" >> "$LOG_FILE"
    # echo "STDERR:" >> "$LOG_FILE"
    # cat "$error_file" >> "$LOG_FILE"

    # 读取错误信息
    error_msg=$(cat "$error_file" | tail -20)

    jq -n --arg code "$exit_code" --arg err "$error_msg" --arg lang "$project_lang" '{
          "decision": "block",
          "reason": ("abcoder parse 失败（语言：\($lang)，退出码: " + $code + "）。错误信息：\n" + $err + "\n\n可能的原因：\n1. 项目配置文件有问题（Go: go.mod；TS: tsconfig.json）\n2. 缺少依赖包\n3. 代码语法错误\n\n建议：\n- Go 项目：运行 'go mod tidy' 和 'go build' 检查\n- TS 项目：运行 'npm install' 和 'tsc --noEmit' 检查"),
          "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "additionalContext": ("解析失败，需要修复后重试")
          }
        }'
  fi

  # 清理临时文件
  trash "$output_file" "$error_file"
else
  # echo "Path did not match" >> "$LOG_FILE"
  echo '{}'
fi

exit 0
