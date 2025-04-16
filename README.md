<!--
 Copyright 2025 CloudWeGo Authors
 
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
 
     https://www.apache.org/licenses/LICENSE-2.0
 
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
-->

# ABCoder: AI-Based Coder(AKA: A Brand-new Coder)

![ABCoder](images/ABCoder.png)

ABCoder, an AI-oriented code handling tool, is designed to enhance coding-context for Large-Language-Model (LLM).


## Features

-  Universal Abstract Syntax Tree (UniAST), an language-independent and AI-friendly  coding-context, provides ample and recursive code information for AI or programs.
  
-  Universal Parser, parses abitary languages to UniAST.

-  Univeral Writer, transforms UniAST back to codes.

- (WIP) Code Understanding and Semantic Querying, which can be used to retrieve codes with natural language for either human or AI.
  
Based on these features, ABCoder can help developers to easily implement or enhance many AI-assisted coding applications, such as code reviewer, IDE copilot and so on.

## Getting Started

1. Install ABCoder:
```bash
go install github.com/cloudwego/abcoder@latest
```
2. Use ABCoder to parse a repository to UniAST (JSON)
```bash
abcoder parse <language> <repo-path> > <AST-path>
```
3. Use ABCoder as a writer
```bash
abcoder write <AST-path>
```

## Universal-Abstract-Syntax-Tree Specification

see [UniAST Specification](docs/uniast-zh.md)


## Supported Languages

ABCoder currently supports the following languages:

| Language | Parser | Writer |
| -------- | ------ | ------ |
| Go       | ✅      | ✅      |
| Rust     | ✅      | WIP    |
| C        | WIP    | ❌      |


## Getting Involved

We encourage developers to contribute and make this tool more powerful. If you are interested in contributing to ABCoder
project, kindly check out our Getting Involved Guide:
- [Parser Extension](docs/parser_extension-zh.md)
- [Writer Extension](docs/writer_extension-zh.md)

> Note: This is a dynamic README and is subject to changes as the project evolves.
