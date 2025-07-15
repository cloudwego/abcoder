# ABCoder: AI-Based Coder(AKA: A Brand-new Coder)

![ABCoder](images/ABCoder.png)

# Overview
ABCoder, an general AI-oriented code-processing SDK, is designed to enhance coding context for Large-Language-Model (LLM), and boost developing AI-assisted-programming applications. 


## Features

-  Universal Abstract-Syntax-Tree (UniAST), an language-independent, AI-friendly specification of code information, providing a boundless, flexible and structrual coding context for both AI and hunman.
  
-  General Parser, parses abitary-language codes to UniAST.

-  General Writer, transforms UniAST back to codes.
  
- (Comming Soon) General Iterator, a framework for visiting the UniAST and implementing code-batch-processing workflows.

- (Comming Soon) Code Retrieval-Augmented-Generation (RAG), provides a set of tools and functions to help the LLM understand your codes much deeper than ever.

Based on these features, developers can easily implement or enhance their AI-assisted-programming applications, such as reviewing, optimizing, translating, etc.


## Universal Abstract-Syntax-Tree Specification

see [UniAST Specification](docs/uniast-zh.md)


# Getting Started

1. Install ABCoder:
```bash
go install github.com/cloudwego/abcoder@latest
```
2. Use ABCoder to parse a repository to UniAST (JSON)
```bash
abcoder parse {language} {repo-path} > ast.json
```
3. Do your magic with UniAST...
4. Use ABCoder to write an UniAST back to codes
```bash
abcoder write {language} ast.json
```


# Supported Languages

ABCoder currently supports the following languages:

| Language | Parser      | Writer      |
| -------- | ----------- | ----------- |
| Go       | ✅           | ✅           |
| Rust     | ✅           | Coming Soon |
| C        | Coming Soon | ❌           |
| Python   | Coming Soon | ❌           |



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