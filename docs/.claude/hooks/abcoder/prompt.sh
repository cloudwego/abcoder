#!/bin/zsh

# 添加调试信息
# LOG_FILE="/tmp/claude-hook-debug.log"

# 错误处理
set -euo pipefail

# 验证文件存在
SOP_FILE=".claude/hooks/abcoder/abcoder-workflow.md"
# echo "DEBUG: Checking file: $SOP_FILE" >&2

if [[ ! -f "$SOP_FILE" ]]; then
  # echo "DEBUG: File not found" >&2
  echo '{"decision": "block", "reason": "SOP file not found", "hookSpecificOutput": {"hookEventName": "PostToolUse"}}'
  exit 0
fi

# echo "DEBUG: File found, reading content" >&2

# 读取并转义内容
SOP_CONTENT=$(cat "$SOP_FILE" | jq -Rs . 2>/dev/null)
if [[ $? -ne 0 ]]; then
  # echo "DEBUG: jq failed" >&2
  echo '{"decision": "block", "reason": "Failed to process SOP content", "hookSpecificOutput": {"hookEventName": "PostToolUse"}}'
  exit 0
fi

# echo "DEBUG: Content processed successfully" >&2

# 输出 JSON
cat <<EOF
{
  "decision": "block",
  "reason": $SOP_CONTENT,
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse"
  }
}
EOF

# echo "DEBUG: Script completed" >&2
exit 0
