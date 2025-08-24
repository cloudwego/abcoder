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

**Package (UNIAST) = TypeScript namespace/module**  
- **Directory**: Source directory with TypeScript files
- **PkgPath**: Import path (e.g., "@myorg/mypackage/utils", "./src/models")
- **IsMain**: True for main entry point (index.ts)
- **IsTest**: True for test files (*.test.ts, *.spec.ts)

### 2. Mapping Strategy

```
Repository (root)
├── Module (npm package)
│   ├── package.json
│   ├── src/ (Package)
│   │   ├── index.ts (IsMain=true)
│   │   ├── utils.ts (PkgPath="./src/utils")
│   │   └── models/
│   │       └── user.ts (PkgPath="./src/models/user")
│   └── test/ (Package)
│       └── index.test.ts (IsTest=true)
└── node_modules/
    ├── lodash (Module - external)
    │   └── ...
    └── react (Module - external)
        └── ...
```

### 3. Package Path Resolution

**Internal Packages**:
- Relative to module root
- Examples: "./src/utils", "./lib/helpers", "./test/mocks"

**External Packages**:
- From node_modules
- Examples: "lodash", "react", "@types/node"

### 4. File to Package Assignment

Each TypeScript file belongs to a Package based on:
1. **Directory structure** - Files in same directory = same package
2. **Import statements** - Used to determine package boundaries
3. **package.json exports** - Define public API boundaries

### 5. Special Cases

**Monorepos**:
- Each workspace = separate Module
- packages/app1, packages/lib1 = separate Modules

**Scoped packages**:
- @myorg/package1 = Module with name "@myorg/package1"
- @myorg/package2 = separate Module

**TypeScript path mapping**:
- tsconfig.json paths affect PkgPath resolution
- "@/*": ["src/*"] → PkgPath="@/utils" maps to "./src/utils"