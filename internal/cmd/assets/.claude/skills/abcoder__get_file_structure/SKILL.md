---
name: skill__abcoder__get_file_structure
description: skill__abcoder__get_file_structure `abcoder cli get_file_structure 'repo_name' 'file_path'` [STRUCTURE] Step 3/4: Get available symbol names of a file. Input: repo_name, file_path from `tree_repo` output. Output: symbol names with signatures. You MUST call `get_file_symbol` later.
---

Execute the get_file_structure command to examine file-level nodes:

```bash
abcoder cli get_file_structure 'repo_name' 'file_path'
```

**Expected Output:**
- Complete node list with type, signature, line
- Imports for the file
- Node IDs for detailed analysis

**Parameters:**
- `repo_name` (required): Repository name from `list_repos`
- `file_path` (required): Relative file path from `get_repo_structure`
```
{
  "description": "[STRUCTURE] level3/4: Get file structure with node list. Input: repo_name, file_path from get_repo_structure output. Output: nodes with signatures.",
  "inputSchema": {
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "properties": {
      "repo_name": {
        "type": "string",
        "description": "the name of the repository (output of list_repos tool)"
      },
      "file_path": {
        "type": "string",
        "description": "relative file path (output of get_repo_structure tool"
      }
    },
    "additionalProperties": false,
    "type": "object",
    "required": [
      "repo_name",
      "file_path"
    ]
  },
  "name": "get_file_structure"
}
```

This tool is **Level 3** in the ABCoder discovery hierarchy. Next: Use [`skill__abcoder__get_file_symbol`](~/.claude/skills/skill__abcoder__get_file_symbol/SKILL.md) to get detailed code information.
