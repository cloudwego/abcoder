---
name: skill__abcoder__search_symbol
description: skill__abcoder__search_symbol `abcoder cli search_symbol <repo_name> <pattern> [--path <path>]` Search symbols in a repository by name pattern. Supports substring match, prefix match (pattern*), suffix match (*pattern), wildcard (*pattern*), and path prefix filtering (--path). You MUST call `get_file_symbol` later.
---

Execute the search_symbol command to search symbols by name:

```bash
abcoder cli search_symbol <repo_name> <pattern> [--path <path>]
```

## Arguments

| Argument | Description |
|----------|-------------|
| `repo_name` | Repository name |
| `pattern` | Symbol name pattern (supports *, regex) |
| `--path` | (optional) Filter by file path prefix (e.g., `src/main/java/com/uniast/parser`) |

## Examples

```bash
# Substring match - search for "Get" in all symbol names
abcoder cli search_symbol myrepo Get

# Prefix match - search for symbols starting with "Get"
abcoder cli search_symbol myrepo "Get*"

# Suffix match - search for symbols ending with "User"
abcoder cli search_symbol myrepo "*User"

# Wildcard match - search for symbols containing "GetUser"
abcoder cli search_symbol myrepo "*GetUser*"

# Path filter - search symbols in specific directory
abcoder cli search_symbol myrepo "Graph" --path "src/main/java/com/uniast/parser"
```

## Output Format

```json
{
  "repo_name": "string",
  "pattern": "string",
  "results": {
    "file_path": {
      "FUNC": ["function_name1", "function_name2"],
      "TYPE": ["type_name"],
      "VAR": ["var_name"]
    }
  }
}
```

## Notes

A powerful search tool based on ABCoder

  Usage: - ALWAYS use `abcoder__search_symbol` for search tasks. NEVER invoke `grep` or `rg` as a Bash command. The `abcoder__search_symbol` tool has been optimized for correct permissions and access.
  - Supports full regex syntax (e.g., "Get*", "Domain*Controller")
  - Dynamic patterns for open-ended searches requiring multiple rounds
  - Pattern syntax: Uses ripgrep (not grep) - literal braces need escaping (use `interface\{\}` to find `interface{}` in Go code)
