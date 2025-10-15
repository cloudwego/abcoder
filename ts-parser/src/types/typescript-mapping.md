# TypeScript Module/Package Mapping Strategy for UNIAST v0.1.3

## TypeScript Structure Analysis

### 1. Module vs Package in TypeScript Context

**Module (UNIAST) = npm package (TypeScript)**
- **Directory**: Contains `package.json`
- **Language**: Always '' (empty string) for TypeScript
- **Name**: From `package.json` name field
- **Version**: From `package.json` version field
- **Dir**: Relative path from repository root
- **Dependencies**: From `package.json` dependencies/devDependencies

**Package (UNIAST) = Individual TypeScript/JavaScript file**
- **File-based**: Each TypeScript/JavaScript file is a separate Package
- **PkgPath**: File path relative to module root (e.g., "src/utils.ts", "src/models/user.ts")
- **IsMain**: True for main entry point files (index.ts, main.ts)
- **IsTest**: True for test files (*.test.ts, *.spec.ts, files in test/ or __tests__/)

### 2. Mapping Strategy

```
Repository (root)
├── Module (npm package)
│   ├── package.json
│   ├── src/
│   │   ├── index.ts (Package: PkgPath="src/index.ts", IsMain=true)
│   │   ├── utils.ts (Package: PkgPath="src/utils.ts")
│   │   └── models/
│   │       └── user.ts (Package: PkgPath="src/models/user.ts")
│   └── test/
│       └── index.test.ts (Package: PkgPath="test/index.test.ts", IsTest=true)
└── node_modules/
    ├── lodash (Module - external)
    │   └── ...
    └── react (Module - external)
        └── ...
```

### 3. Package Path Resolution

**Internal Packages**:
- File path relative to module root
- Examples: "src/utils.ts", "lib/helpers.ts", "test/mocks.ts"

**External Packages**:
- From node_modules, using module name
- Examples: "lodash", "react", "@types/node"

### 4. File to Package Assignment

**One File = One Package**:
1. **File-level granularity** - Each TypeScript/JavaScript file constitutes a separate Package
2. **PkgPath** - File's relative path from module root (with forward slashes on all platforms)
3. **No directory grouping** - Files are not grouped by directory into packages
4. **Independent parsing** - Each file is parsed independently as its own Package

### 5. Special Cases

**Monorepos**:
- Each workspace = separate Module
- packages/app1, packages/lib1 = separate Modules
- Each file within a workspace is a separate Package

**Scoped packages**:
- @myorg/package1 = Module with name "@myorg/package1"
- @myorg/package2 = separate Module
- Files within scoped packages follow the same file-level Package rule

**TypeScript path mapping**:
- tsconfig.json paths affect module resolution during parsing
- Actual PkgPath remains the physical file path relative to module root
- Path aliases are resolved during symbol resolution, not for PkgPath naming