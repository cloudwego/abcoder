---
name: skill__abcoder__tree_repo
description: skill__abcoder__tree_repo `abcoder cli tree_repo 'repo_name' [DISCOVERY] Level 2/4: [STRUCTURE] Step 2/4: Get available file_paths of a repo. Input: repo_name from `list_repos` output. Output: available file_paths. You MUST call `get_file_structure` later.
---

Execute the tree_repo command to examine repository-level structure:

```bash
abcoder cli tree_repo 'repo_name'
```

**Expected Output:**
- Complete repository file paths

**Parameters:**
- `repo_name` (required): Repository name from `list_repos` output
