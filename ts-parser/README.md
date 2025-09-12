# TypeScript Parser for ABCoder

A TypeScript AST parser that extracts method calls, variable references, and dependencies.

Usage: 

Build: `npm run build`

Run: `node dist/index.js parse [options] <directory>`

Parse a TypeScript repository and generate UNIAST JSON

Arguments:
  directory              Directory to parse


| Option | Description |
|--------|-------------|
| -o, --output <file> | Output file path (default: "output.json") |
| -t, --tsconfig <file> | Path to tsconfig.json file, if you provide a relative path, it will be relative to **the directory of the input file** (default: "tsconfig.json") |
| --no-dist | Ignore dist folder and its contents |
| --pretty | Pretty print JSON output |
| --src <dirs> | Directory paths to include (comma-separated) |
| -h, --help | display help for command |


See `./index.ts` for more information.


## Notes

1. MUST correctly specify the location of the current project's `tsconfig.json`. 

2. If you provide a relative path to argument `--tsconfig`, it will be relative to **the directory of the input file**.

3. Before usage, please configure the dependencies for your TypeScript project, such as running npm install and setting up cross-package dependencies in monorepo.

4. If the repository you're analyzing is too large, you may need to adjust Node.js's maximum memory allocation.

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