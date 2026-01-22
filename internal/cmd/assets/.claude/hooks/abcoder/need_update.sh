#!/bin/bash

# 当 Claude Code 执行 Write 操作后，更新 _need_update 文件为 1

# 错误处理
set -euo pipefail

# 获取脚本所在的绝对路径
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 定义 _need_update 文件路径（放在脚本所在目录）
need_update_file="${script_dir}/_need_update"

# 写入 1 到文件
echo "1" > "$need_update_file"

# 输出 JSON
jq -n --arg file "$need_update_file" '{
  "continue": true,
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": ("Write 操作完成，已更新 " + $file + " 为 1")
  }
}'

exit 0
