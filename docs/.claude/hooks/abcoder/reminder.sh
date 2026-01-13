#!/bin/zsh
cat <<EOF
{
  "continue": true,
  "systemMessage": "<system-reminder>This is a reminder that when executing the ABCoder code analysis workflow, after locating the target node, you MUST use the get_ast_node tool. It is required to recursively call get_ast_node to obtain the complete AST node information, including type, code, position, and related relationships (dependency, reference, inheritance, implementation, grouping node IDs).</system-reminder>",
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse"
  }
}
EOF
exit 0
