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

ABCoder, an AI-powered tool, streamlines coding by keeping real-time status updates, providing lossless code compression, and giving development guidance. It enhances testing by identifying quality, generating reports, and auto-creating test cases. It also offers guidance for refactoring, including language stack switches.

# Table of Contents

- [ABCoder: AI-Based Coder(AKA: A Brand-new Coder)](#abcoder-ai-based-coderaka-a-brand-new-coder)
- [Table of Contents](#table-of-contents)
- [Overview](#overview)
- [Quick Start](#quick-start)
  - [Prerequisites](#prerequisites)
  - [Running through Coze OpenAPI](#running-through-coze-openapi)
- [Status Update](#status-update)
- [Lossless Compression](#lossless-compression)
- [Development Guide](#development-guide)
- [Testing Enhancements](#testing-enhancements)
- [Refactor/Rewrite Guide](#refactorrewrite-guide)
- [Getting Involved](#getting-involved)

# Overview

ABCoder is a comprehensive open-source software development tool that aims to utilize artificial intelligence to enhance
the process of coding. This project focuses on various aspects of software development ranging from repository analysis,
issue and pull request tracking, to automated code compression, development guidance, testing enhancement, and
refactoring guidance.

# Quick Start

## Prerequisites
- install git and set your access token for github on cmd-line
- install [rust-toolchain](https://www.rust-lang.org/tools/install) (stable)
- (optional) install [ollama](https://github.com/ollama/ollama) and run your LLM
- (optional) create a [Coze](https://www.coze.com/docs/developer_guides/coze_api_overview?_lang=en) agent and set its OpenAPI key

## Running through Coze OpenAPI
1. Set .env file for configuration on ABCoder's working directory. Taking Coze as an example:
```
# cache for repo，AST and so on
WORK_DIR=tmp_abcoder

# exclude dirs for repo parsing, separated by comma
EXCLUDE_DIRS=target,gen-codes 

# LLM's api type
API_TYPE=coze # coze|ollama 

# LLM's output language
LANGUAGE=zh 

# Coze options
COZE_API_TOKEN="{YOUR_COZE_API_TOKEN}"
COZE_BOT_ID={YOUR_COZE_BOT_ID}
```

2. compile the parsers
```
sh ./script/make_parser.sh
```

3. compile and run ABCoder
```
cargo run --bin cmd compress https://xxx.git
```

4. Once triggered, ABCoder will take three steps:
   1. Download the repository in {REPO_DIR}
   2. Parse the repository and store the AST in {CACHE_DIR}
   3. Call the LLM to compress the repository codes, and refresh the AST for each call.
You can stop the process at anytime after step 2. You can restart the compressing by running the same command.

5. Export the compressed results
```
cargo run --bin cmd export https://xxx.git --out-dir {OUTPUT_DIR}
```

# Status Update

The system is designed to automatically fetch the latest data from Github upon triggering relevant tasks, ensuring the
repository status is always up-to-date. It can answer queries related to function, defects based on issue and PR
information. For more details, check out our Issues and Pull Requests sections on Github.

# Lossless Compression

The system also offers a lossless compression feature for repository code. The specific implementation methods are being
optimized, and more details will be available soon.

# Development Guide

We welcome all developers wishing to contribute to ABCoder. Our system provides detailed guidance for manual development
and also supports auto-generation of instructions. Check out our Contribution Guide for more information.

# Testing Enhancements

The system is designed to analyze existing functions and corresponding tests, identify the overall quality of testing,
produce reports, and automatically generate test cases for weakly covered items. Our goal is to help repositories
enhance and perfect their test cases.

# Refactor/Rewrite Guide

We offer guidance for both small-scale feature iterations and large-scale rewrites, including language stack switches.
Our system provides a detailed guide for manual development and also supports automated guidance generation.

# Getting Involved

We encourage developers to contribute and make this tool more powerful. If you are interested in contributing to ABCoder
project, kindly check out our Getting Involved Guide.

> Note: This is a dynamic README and is subject to changes as the project evolves.
