# Java 解析环境搭建指南

本文档将指导您如何配置本地环境以使用 `abcoder` 的 Java 解析功能。

为了成功解析 Java 项目，`abcoder` 依赖于以下两个核心组件：
1.  **Java 运行环境 (JRE)**: 版本需要 **17 或更高**。
2.  **Eclipse JDT Language Server**: 一个用于代码分析的语言服务器。

我们提供了一个自动化脚本来简化安装过程，同时也支持手动配置以满足个性化需求。

## 1. 快速开始 (推荐)

我们强烈建议您使用提供的脚本来自动完成环境的搭建。

### 步骤 1: 授予脚本执行权限

在项目根目录下，执行以下命令：

```bash
chmod +x lang/java/lsp/setup_java_env.sh
```

### 步骤 2: 运行设置脚本

执行此脚本会自动检查您的 Java 版本、下载并解压 JDT Language Server，并为当前终端会话设置必要的环境变量 (`LAUNCHER_JAR` 和 `JDTLS_ROOT_PATH`)。

**重要**: 您需要使用 `source` 命令来执行脚本，以确保环境变量在当前会话中生效。

```bash
source lang/java/lsp/setup_java_env.sh
```

脚本执行成功后，您的环境就准备就绪了。

## 2. 运行 Java 解析器

环境配置完成后，您可以使用 `parse` 命令来解析您的 Java 项目。

### 命令格式

```bash
go run . parse java <待解析的Java项目路径> -o <输出的JSON文件路径>
```

### 示例

例如，解析位于 `/Users/bytedance/Documents/code/travel-auth` 的项目，并将结果输出到 `testdata/tmp/java.json`：

```bash
go run . parse java /Users/bytedance/Documents/code/travel-auth -o /Users/bytedance/GolandProjects/abcoder/testdata/tmp/java.json
```

## 3. 手动配置与自定义 (高级)

如果您希望使用自定义的 Java 安装或 JDT Language Server 路径，可以跳过自动化脚本，进行手动配置。

### a. 自定义 Java Home

如果您的 Java 17+ 安装在非标准路径，或者您希望明确指定一个 Java 版本，可以在执行 `parse` 命令时通过 `--java-home` 标志来指定。

```bash
go run . parse java <项目路径> --java-home /path/to/your/java_home
```

### b. 自定义 JDT Language Server 路径

如果您已经手动下载或在其他位置安装了 JDT Language Server，您只需在运行 `parse` 命令前，设置 `JDTLS_ROOT_PATH` 环境变量，使其指向 JDT Language Server 的根目录即可。

例如：

```bash
# 设置环境变量
export JDTLS_ROOT_PATH=/path/to/your/jdt-language-server-x.y.z

# 运行解析命令
go run . parse java <项目路径> -o <输出路径>
```

`abcoder` 会优先使用 `JDTLS_ROOT_PATH` 环境变量指定的路径来启动语言服务器。

---

现在，您可以根据自己的需求选择快速或手动方式来配置环境，并开始使用强大的 Java 解析功能了。