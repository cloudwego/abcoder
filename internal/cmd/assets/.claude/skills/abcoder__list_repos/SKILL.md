---
name: skill__abcoder__list_repos
description: skill__abcoder__list_repos `abcoder cli list_repos` [DISCOVERY] Step 1/4: List available repositories. Always the first step in ABCoder workflow. You MUST call `tree_repo` later.
---

Execute the list_repos command to discover all available repositories:

```bash
abcoder cli list_repos
```

## Workflow Context

This tool is **Level 1** in the 4-level ABCoder discovery hierarchy:

1. **Level 1 (This Tool)**: `list_repos` - List all repositories
2. **Level 2**: `tree_repo` - Get repository structure
3. **Level 3**: `get_file_structure` - Get file nodes details
4. **Level 4**: `get_file_symbol` - Get detailed AST node information

## Usage Pattern
Output
```
{
  "repo_names": {
    "array[i]": "string"
  }
}
```
