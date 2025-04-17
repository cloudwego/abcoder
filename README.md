# ABCoder: AI-Based Coder(AKA: A Brand-new Coder)

![ABCoder](images/ABCoder.png)

ABCoder, an AI-oriented code-processing SDK, is designed to enhance coding context for Large-Language-Model (LLM), and boost developing AI-assisted-coding workflow.

## Features

-  Universal Abstract Syntax Tree (UniAST), an language-independent, AI-friendly code-struct specfication, providing flexible and structrual coding-context for both AI and hunman.
  
-  Universal Parser, parses abitary languages to UniAST.

-  Univeral Writer, transforms UniAST back to codes.
  
- (Comming Soon) Univeral Iterator, provides a set of interfaces and tools to help developers to implement their agents without deep knowledge of the UniAST structure.

- (Comming Soon) Code RAG, provides a set of tools to help the LLM understand your codes much deeper than ever.

Based on these features, developers can easily implement or enhance their AI-assisted-coding workflows (or agents), such as reviewing, optimizing, translating...

## Getting Started

1. Install ABCoder:
```bash
go install github.com/cloudwego/abcoder@latest
```
2. Use ABCoder to parse a repository to UniAST (JSON)
```bash
abcoder parse {language} {repo-path} > ast.json
```
3. Do your magic with UniAST...
4. Use ABCoder to write a UniAST back to codes
```bash
abcoder write {language} ast.json
```

## Universal-Abstract-Syntax-Tree Specification

see [UniAST Specification](docs/uniast-zh.md)


## Supported Languages

ABCoder currently supports the following languages:

| Language | Parser      | Writer      |
| -------- | ----------- | ----------- |
| Go       | ✅           | ✅           |
| Rust     | ✅           | Coming Soon |
| Kotlin   | Coming Soon | ❌           |
| C        | WIP         | ❌           |


## Getting Involved

We encourage developers to contribute and make this tool more powerful. If you are interested in contributing to ABCoder
project, kindly check out our Getting Involved Guide:
- [Parser Extension](docs/parser-zh.md)

> Note: This is a dynamic README and is subject to changes as the project evolves.
