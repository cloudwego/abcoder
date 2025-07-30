# Language Server Installation

To parse dependencies between symbols in a repository, the abcoder parser requires the use of language servers for various languages. Please install the corresponding language server before running the parser.

The mapping between languages and language servers is as follows:

| Language | Language Server                        | Executable      |
| -------- | -------------------------              | --------------- |
| Go       | Does not use LSP, uses built-in parser | /               |
| Rust     | rust-analyzer                          | rust-analyzer   |
| Python   | (Modified) python-lsp-server           | pylsp           |
| C        | clangd-18                              | clangd-18       |

Ensure the corresponding executable is in PATH before running abcoder.

## Rust
* First, install the Rust language via [rustup](https://www.rust-lang.org/tools/install).
* Install rust-analyzer:
  ```bash
  $ rustup component add rust-analyzer
  $ rust-analyzer --version # Verify successful installation
  ```

## Python
* Install Python 3.9+
* Install pylsp from the git submodule:
  ```bash
  $ git submodule init
  $ git submodule update
  $ cd pylsp
  $ pip install -e . # Consider executing in a separate conda/venv environment
  $ export PATH=$(realpath ./bin):$PATH # Add this to your .rc file, or set it before each abcoder run
  $ pylsp --version # Verify successful installation
  ```

## C
* Ubuntu 24.04 or later: Install directly from apt:
  ```bash
  $ sudo apt install clangd-18
  ```

* Other distributions: Use a manual installation.
  Or download a pre-compiled version from the [LLVM official website](https://releases.llvm.org/download.html). clangd is in `clang-tools-extra`.
