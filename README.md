# ABCoder: AI-Based Coder(AKA: A Brand-new Coder)

![ABCoder](images/ABCoder.png)

# Overview
ABCoder, an general AI-oriented code-processing SDK, is designed to enhance coding context for Large-Language-Model (LLM), and boost developing AI-assisted-programming applications. 


## Features

-  Universal Abstract-Syntax-Tree (UniAST), an language-independent, AI-friendly specification of code information, providing a boundless, flexible and structrual coding context for both AI and hunman.
  
-  General Parser, parses abitary-language codes to UniAST.

-  General Writer, transforms UniAST back to codes.

- Code-Retrieval-Augmented-Generation (Code-RAG), provides a set of MCP tools to help the LLM understand your codes more precisely.

Based on these features, developers can easily implement or enhance their AI-assisted-programming applications, such as reviewing, optimizing, translating, etc.


## Universal Abstract-Syntax-Tree Specification

see [UniAST Specification](docs/uniast-zh.md)


# Quick Start

Below is a quick start guide for using ABCoder to build a coding context on both internal and external libraies.

1. Install ABCoder:

    ```bash
    go install github.com/cloudwego/abcoder@latest
    ```

2. Use ABCoder to parse a repository to UniAST (JSON)

    ```bash
    abcoder parse {language} {repo-path} > xxx.json
    ```

    for example:

    ```bash
    git clone https://github.com/cloudwego/localsession.git localsession
    abcoder parse go localsession -o /abcoder-asts/localsession.json
    ```

3. Integrate ABCoder's MCP tools into your AI agent.

    ```json
    {
        "mcpServers": {
            "abcoder": {
                "command": "abcoder",
                "args": [
                    "mcp",
                    "{the-AST-directory}" // EX: "/abcoder-asts"
                ]
            }
        }
    }
    ```


4. Enjoy it!

    See [using ABCoder in TRAE](https://bytedance.sg.larkoffice.com/file/SEmdbLpC1oCbclxmc5Dlkp9fg7r). Tips:

    - You can add more repo ASTs into the AST directory without restarting abcoder MCP server.
    
    - Try to use [the recommaned prompt](llm/prompt/analyzer.md) and combine planning/memory tools like [sequential-thinking](https://github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking) in your AI agent.


# Supported Languages

ABCoder currently supports the following languages:

| Language | Parser      | Writer      |
| -------- | ----------- | ----------- |
| Go       | ✅           | ✅           |
| Rust     | ✅           | Coming Soon |
| C        | ✅           | ❌           |
| Python   | Coming Soon | Coming Soon |


# Getting Involved

We encourage developers to contribute and make this tool more powerful. If you are interested in contributing to ABCoder
project, kindly check out our guide:
- [Parser Extension](docs/parser-zh.md)

> Note: This is a dynamic README and is subject to changes as the project evolves.


# Contact Us
- How to become a member: [COMMUNITY MEMBERSHIP](https://github.com/cloudwego/community/blob/main/COMMUNITY_MEMBERSHIP.md)
- Issues: [Issues](https://github.com/cloudwego/abcoder/issues)
- Lark: Scan the QR code below with [Register Feishu](https://www.feishu.cn/en/) to join our CloudWeGo/abcoder user group.

&ensp;&ensp;&ensp; <img src="images/lark_group_zh.png" alt="LarkGroup" width="200"/>


# License
This project is licensed under the [Apache-2.0 License](LICENSE-APACHE).