# Language Server Installation
To parse dependencies between symbols in a repository, the abcoder parser requires the use of language servers for various languages. Please install the corresponding language server before running the parser.
ABCoder automatically installs the corresponding language server for most languages.
Should the automatic installation fail, please install manually following the instructions below.

The mapping between languages and language servers is as follows:

| Language   | Language Server                                                    | Essential Environment |
| ---------- | ------------------------------------------------------------------ | --------------------- |
| Go         | NA                                                                 | golang 1.23+          |
| TypeScript | NA                                                                 | node.js 20+           |
| Rust       | rust-analyzer (official)                                           | rust-toolchain        |
| Python     | pylsp ([modified](https://github.com/Hoblovski/python-lsp-server)) | Python 3.9+           |
| C          | clangd-18 (official)                                               | clang 18+             |
| Java       | eclipse-jdtls (official)                                           | java 17+              |

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
* Install pylsp
  ```bash
  $ git clone https://github.com/Hoblovski/python-lsp-server.git -b abc
  $ cd python-lsp-server
  $ pip install .
  $ export PATH=$(realpath ./bin):$PATH
  $ pylsp --version
  ```

## C
* Ubuntu 24.04 or later: Install directly from apt:
  ```bash
  $ sudo apt install clangd-18
  ```

* Other distributions: Use a manual installation.
  Or download a pre-compiled version from the [LLVM official website](https://releases.llvm.org/download.html). clangd is in `clang-tools-extra`.
