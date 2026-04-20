---
name: skill__abcoder__get_file_symbol
description: skill__abcoder_get_file_symbol `abcoder cli get_file_symbol 'repo_name' 'relative_file_path' 'symbol_name'` [ANALYSIS] Step 4/4: Get symbol's code, dependencies and references; use refer/depend's file_path and name as next `get_file_symbol` input. Input: repo_name, file_path, name. Output: codes, dependencies, references. You MUST call `get_file_symbol` with refers/depends file_path and name to check its code, call-chain or data-flow detail.
---

Execute the get_file_symbol command to get detailed symbol information:

```bash
abcoder cli get_file_symbol 'repo_name' 'relative_file_path' 'symbol_name'
```

**Expected Output:**
```
{
  "nodes": [
    {
      "file_path": "string",
      "name": "string",
      "type": "string",
      "line": "number",
      "codes": "string",
      "dependencies": [
        {
          "file_path": "string",
          "names": ["string"]
        }
      ],
      "references": [
        {
          "file_path": "string",
          "names": ["string"]
        }
      ]
    }
  ]
}
```

**Parameters:**
- `repo_name` (required): Repository name from `list_repos`
- `file_path` (required): File path from `get_repo_structure`
- `symbol_name` (required): Name of the symbol to query
```
{
  "description": "[ANALYSIS] level4/4: Get detailed AST node info by file path and symbol name. Output: codes, dependencies, references, implementations (all grouped by file_path).",
  "inputSchema": {
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "properties": {
      "repo_name": {
        "type": "string",
        "description": "the name of the repository (output of list_repos tool)"
      },
      "file_path": {
        "type": "string",
        "description": "the file path (output of get_repo_structure tool)"
      },
      "symbol_name": {
        "type": "string",
        "description": "the name of the symbol (function, type, or variable) to query"
      }
    },
    "additionalProperties": false,
    "type": "object",
    "required": [
      "repo_name",
      "file_path",
      "symbol_name"
    ]
  },
  "name": "get_file_symbol"
}
```

**Recursive Analysis:** Use this tool recursively to trace code calling chains. From the `dependencies` and `references` arrays, extract the `file_path` and `symbol_name` for related nodes, then call `get_file_symbol` again to dive deeper into the calling chain. 
