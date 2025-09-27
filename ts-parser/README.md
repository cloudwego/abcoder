# TypeScript Parser for ABCoder

A TypeScript AST parser that extracts method calls, variable references, and dependencies with advanced monorepo support and intelligent parsing strategies.

## Features

- 🚀 **Monorepo Support**: Intelligent detection and parsing of monorepo projects
- ⚡ **Smart Parsing Strategy**: Automatic selection between single-process and cluster-based parsing
- 📦 **Multiple Monorepo Formats**: Support for Edex, pnpm workspaces, Lerna
- 🎯 **Flexible Output Modes**: Combined or separate repo output for monorepo packages

## Usage

Build: `npm run build`

Run: `node dist/index.js parse [options] <directory>`

Parse a TypeScript repository and generate UNIAST JSON

Arguments:
directory Directory to parse

## Examples

### Basic Usage

- **Parse a single TypeScript project** : `node dist/index.js parse ./my-project`

- **Parse with pretty output** : `node dist/index.js parse ./my-project --pretty`

- **Parse monorepo with separate package outputs** : `node dist/index.js parse ./my-monorepo --monorepo-mode separate`

- **Parse monorepo (combined output)**: `node dist/index.js parse ./my-monorepo`
- **Parse monorepo (separate output for each package)**: `node dist/index.js parse ./my-monorepo --monorepo-mode separate`

- **Custom output path** : `node dist/index.js parse ./my-project -o ./output/result.json `

## Options

| Option                 | Description                                                                                                                                       |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| -o, --output <file>    | Output file path (default: "output.json")                                                                                                         |
| -t, --tsconfig <file>  | Path to tsconfig.json file, if you provide a relative path, it will be relative to **the directory of the input file** (default: "tsconfig.json") |
| --no-dist              | Ignore dist folder and its contents                                                                                                               |
| --pretty               | Pretty print JSON output                                                                                                                          |
| --src <dirs>           | Directory paths to include (comma-separated)                                                                                                      |
| --monorepo-mode <mode> | Monorepo output mode: "combined" (entire repository) or "separate" (each package) (default: "combined")                                           |
| -h, --help             | display help for command                                                                                                                          |

See `./index.ts` for more information.

## Monorepo Support

The parser automatically detects and supports various monorepo configurations:

- **Eden Monorepo**: Supports both `packages` format and `workspaces` format
- **pnpm Workspaces**: Reads `pnpm-workspace.yaml` configuration
- **Lerna**: Detects `lerna.json` configuration

### Parsing Strategies

The parser intelligently selects the optimal parsing strategy based on project size:

- **Single Process Mode**: For small to medium projects
- **Cluster Mode**: For large projects with parallel processing across multiple CPU cores

## Notes

1. MUST correctly specify the location of the current project's `tsconfig.json`.

2. If you provide a relative path to argument `--tsconfig`, it will be relative to **the directory of the input file**.

3. Before usage, please configure the dependencies for your TypeScript project, such as running npm install and setting up cross-package dependencies in monorepo.

4. For large monorepo projects, the parser will automatically use cluster-based processing to improve performance.

5. If the repository you're analyzing is too large, you may need to adjust Node.js's maximum memory allocation.

## Terminology

**Package vs Module**: In JavaScript/TypeScript terminology, a "Package" typically refers to an npm package (defined by `package.json`). However, in our UniAST output, what JavaScript/TypeScript calls a "Package" corresponds to a "Module" in the UniAST structure. This means:

- **TypeScript/JavaScript Package** (npm package with `package.json`) → **UniAST Module**
- **TypeScript/JavaScript Module** (individual `.ts`/`.js` files) → **UniAST Package**

This terminology mapping is used consistently throughout the parser to align with the UniAST specification, but it may initially seem counterintuitive to developers familiar with JavaScript/TypeScript conventions.

## Some known issues

- When there is a circular dependency, the parser will choose one of the dependencies as the main dependency.
- The parser does not handle dynamic imports.
- The parser does not handle TypeScript decorators.
- For external symbol which has no `.d.ts` declaration file, the parser will not be able to resolve the symbol.
