# Language server 安装
为了解析仓库中符号之间的依赖，abcoder parser 需要使用各语言的 language server。
运行 parser 之前请安装对应的 language server。

语言和 language server 的对应关系如下

| 语言       | Language server                                                        | 必要运行环境   |
| ---------- | ---------------------------------------------------------------------- | -------------- |
| Go         | NA                                                                     | golang 1.23+   |
| TypeScript | NA                                                                     | node.js 20+    |
| Rust       | rust-analyzer (官方)                                                   | rust-toolchain |
| Python     | pylsp (基于官方[修改](https://github.com/Hoblovski/python-lsp-server)) | Python 3.9+    |
| C          | clangd-18 (官方)                                                       | clang 18+      |
| Java       | eclipse-jdtls (官方)                                                   | openjdk 17+    |

按如下教程完成安装后，在运行 abcoder 前请确保 PATH 中有对应可执行文件

## Rust
* 先通过 [rustup](https://www.rust-lang.org/tools/install) 安装 Rust 语言
* 安装 rust-analyzer
  ```bash
  $ rustup component add rust-analyzer
  $ rust-analyzer --version  # 验证安装成功
  ```

## Python
* 安装 Python 3.9+
* 从 git submodule 安装 pylsp
  ```bash
  $ git submodule init
  $ git submodule update
  $ cd pylsp
  $ pip install -e .  # 可以考虑在单独的 conda/venv 环境中执行
  $ export PATH=$(realpath ./bin):$PATH  # 放到 .rc 文件里，或每次运行 abcoder 前都设置一下
  $ pylsp --version  # 验证安装成功
  ```

## C
* ubuntu 24.04 或以后版本: 可以直接从 apt 安装
  ```bash
  $ sudo apt install clangd-18
  ```

* 其他发行版：手动编译、或从 [llvm 官方网站](https://releases.llvm.org/download.html) 下载预编译的版本。
  clangd 在 clang-tools-extra 中。

