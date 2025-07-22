# Universal Abstract-Syntax-Tree Specification (v0.1.2)

Universal Abstract-Syntax-Tree is a LLM-friendly, language-agnostic code context data structure established by ABCoder. It represents a unified abstract syntax tree of a repository's code, collecting definitions of language entities (functions, types, constants/variables) and their interdependencies for subsequent AI understanding and coding-workflow development.


# Identity Node Unique Identification

To ensure precise querying and scalable storage, `ModPath?PkgPath#SymbolName` is约定 as the globally unique identifier for AST Nodes.


```json
{
    "ModPath": "github.com/cloudwego/localsession",
    "PkgPath": "github.com/cloudwego/localsession/backup",
    "Name": "RecoverCtxOnDemands"
}
```

- ModPath: A complete build unit where the content is the installation path@version number. This information is not required for LLMs but is preserved to ensure global uniqueness of Identity. It corresponds to different concepts in various languages:

    - <u>Golang</u>: Corresponds to a module, e.g., github.com/cloudwego/hertz@v0.1.0

    - Rust: Corresponds to a crate, e.g., [serde_json](https://crates.io/crates/serde_json)@v1.0.114

    For easier manual debugging, there's an implementation convention:

    - Module nodes for internal repository functions or types (including functions in this package and other subpackages within the repository) **typically** do not include version numbers

    - Module nodes for external functions (functions imported from third-party repositories) **typically** include version numbers

    - To determine if a Module is a third-party dependency, try to use whether Module.Dir is empty; this is not guaranteed


- PkgPath: An independent namespace in the language, corresponding to the import path of a package in the language

    - Golang: Corresponds to a package, e.g., github.com/cloudwego/hertz/pkg/app/server

    - Rust: Corresponds to a mod, e.g., [serde_json](https://crates.io/crates/serde_json)::[value](https://docs.rs/serde_json/1.0.114/serde_json/value/index.html)

    - Note: This should be as equivalent as possible to the import (use) path in code files for easier LLM understanding


- Name: The unique symbol name within the package
  
    - If the node is a method, it should be represented as `TypeName.MethodName`.
  
    - Additionally, some languages like **Rust allow a type to implement methods with the same name for different interfaces**, so to avoid conflicts, TypeName can be further extended to the `InterfaceName<ImplTypeName>` format


- **String (Key) Format**

    - Full() complete format: `ModPath?PkgPath#Name`

    - String() Format: `PkgPath#Name`, which is generally sufficient for display to LLMs


- **Every AST will have an Identity, but it is embedded in specific node fields (Name, ModPath, PkgPath three fields**


# Go Struct Format
- See [Repository](/lang/uniast/ast.go) for code definition


# JSON Format

The following uses the [cloudwego/localsession](https://github.com/cloudwego/localsession.git) library parsing as an example


## Repository

A repository consists of entity Modules and relationship Graph


```json
{
    "Identity": "/Users/bytedance/golang/work/abcoder/tmp/localsession",
    "Modules": {
        "github.com/bytedance/gopkg@v0.0.0-20230728082804-614d0af6619b": {},
        "github.com/cloudwego/localsession": {}
    },
    "Graph": {}
}
```

- Identity: The unique name of the repo. Since the abcoder parser does not currently retrieve repository git information, the absolute path where it is currently located is generally used as the Identity


- Modules: Contains submodules, a dictionary of {ModPath}: {Module AST}. Both repository modules and external dependency modules can appear in Modules, but need to be distinguished by ModulePath.

    - Repository module: ModePath == Module.Name

    - External dependency module: ModePath == Modele.Name@version


- Graph: Dependency topology graph of AST Nodes, see [Graph] below


### Module

An independent code compilation unit, corresponding to ModPath in Identity, containing various packages


```json
{
    "Name": "github.com/cloudwego/localsession",
    "Language": "go",
    "Version": "",
    "Name": "github.com/cloudwego/localsession",
    "Dir": ".",
    "Packages": {
        "github.com/cloudwego/localsession": {},
        "github.com/cloudwego/localsession/backup": {}
    },
    "Dependencies": {
        "github.com/bytedance/gopkg": "github.com/bytedance/gopkg@v0.0.0-20230728082804-614d0af6619b"
    },
    "Files": {
        ".github/ISSUE_TEMPLATE/bug_report.md": {},
        "backup/metainfo.go": {}
    }
}
```

- Name: Module name (without version number)


- Language: The language used by the code -- For multi-language repositories, a module must have a unique language. However, a repository can have modules in different languages.


- Dir: The relative path of the module to the repository root. Note:

    - **Only modules within this repository need to be set and cannot be empty**

    - **Third-party dependencies must be empty (currently used to determine if it is a third-party dependency)**


- Dependencies: Dictionary of third-party dependency modules for module building {ModName}: {ModPath}


- Packages: Contains subpackages, {PkgPath}: {Pacakge AST} dictionary


- Files: Module file information, where the key is the **path relative to the repo**. It is recommended to include all repository files here to facilitate writer rewriting


#### File

File information, including both code and non-code files


```json

{
    "Path": "manager.go",
    "Imports": [],
    "Package": "github.com/cloudwego/localsession"
}
```

- Path: File **path relative to repository root**


- Imports: import code,


##### Import

```json
{
    "Alias": "_",
    "Path": "\"unsafe\""
}
```

- Path: The import path is mainly used for the writer to write code, and the specific content depends on the language:

    - In Rust: `use xx::yy;`

    - In Golang: `"github.com/cloudwego/abcoder"`


- Alias: Import alias, can be empty


#### Package

A code namespace, corresponding to Identity.PkgPath, containing various AST Node entities


```json
{
    "IsMain": false,
    "IsTest": false,
    "PkgPath": "github.com/cloudwego/localsession/backup",
    "PkgPath": "github.com/cloudwego/localsession/backup",
    "Functions": {
        "BackupCtx": {}
    },
    "Types": {},
    "Vars": {}
}
```

- PkgPath: Module path, see [Identity] introduction


- IsMain: Whether it is a binary package


- IsTest: Whether it is a test package


- Functions: Contains function ASTs, {FuncName}: {Function AST} dictionary


- Types: Contains type ASTs, {TypeName}: {Type AST} dictionary


- Vars: Contains global variables/constants, {VarName}: {Variant AST} dictionary


##### Function

Function type AST Node entity, corresponding to [NodeType] as FUNC, including functions, methods, interface functions


```json
{
    "Exported": true,
    "IsMethod": true,
    "IsInterfaceMethod": false,
    "ModPath": "github.com/cloudwego/localsession",
    "PkgPath": "github.com/cloudwego/localsession",
    "Name": "SessionManager.BindSession",
    "File": "manager.go",
    "Line": 134,
    "StartOffset": 3290,
    "EndOffset": 3573,
    "Content": "// BindSession binds the session with current goroutine\nfunc (self *SessionManager) BindSession(Identity SessionIdentity, s Session) {\n\tshard : = self.shards[uint64(Identity)%uint64(self.opts.ShardNumber)]\n\n\tshard.Store(Identity, s)\n\n\tif self.opts.EnableImplicitlyTransmitAsync {\n\t\ttransmitSessionIdentity(Identity)\n\t}\n}",
    "Signature": "func (self *SessionManager) BindSession(Identity SessionIdentity, s Session)",
    "Receiver": {
        "IsPointer": true,
        "Type": {
            "ModPath": "github.com/cloudwego/localsession",
            "PkgPath": "github.com/cloudwego/localsession",
            "Name": "SessionManager"
        }
    },
    "Params": [
        {
            "ModPath": "github.com/cloudwego/localsession",
            "PkgPath": "github.com/cloudwego/localsession",
            "Name": "SessionIdentity",
            "File": "manager.go",
            "Line": 134,
            "StartOffset": 3386,
            "EndOffset": 3398
        },
        {
            "ModPath": "github.com/cloudwego/localsession",
            "PkgPath": "github.com/cloudwego/localsession",
            "Name": "Session",
            "File": "manager.go",
            "Line": 134,
            "StartOffset": 3400,
            "EndOffset": 3409
        }
    ],
    "FunctionCalls": [
        {
            "ModPath": "github.com/cloudwego/localsession",
            "PkgPath": "github.com/cloudwego/localsession",
            "Name": "transmitSessionIdentity",
            "File": "manager.go",
            "Line": 140,
            "StartOffset": 3547,
            "EndOffset": 3564
        }
    ],
    "MethodCalls": [
        {
            "ModPath": "github.com/cloudwego/localsession",
            "PkgPath": "github.com/cloudwego/localsession",
            "Name": "com/cloudwego/localsession.Store",
            "File": "manager.go",
            "Line": 137,
            "StartOffset": 3485,
            "EndOffset": 3490
        }
    ],
    "Types": [],
    "Vars": []
}
```

- ModPath: Module path, see [Identity] introduction


- PkgPath: Package path, see [Identity] introduction


- Name: Function name

    - If the function is a method, it **should** be represented as {TypeName}.{Methodname}


- File: The filename where it is located


- Line: **Line number of the starting position in the file (starting from 1)**


- StartOffset: **Byte offset of the code starting position relative to the file header** 


- EndOffset: **Byte offset of the code ending position relative to the file header** 


- Exported: Whether visible/exported outside the package


- IsMethod: Whether it is a method

- Signature: Function signature, including function name, parameters, return values, etc.

- IsInterfaceMethod: Whether it is an interface method -- Here abcoder parse collects InterfaceMethod for easier LLM understanding, but it is not considered a language entity in write


- Receiver: If it is a method, there will be a receiver struct.

    - IsPointer: Whether it is a pointer receiver (can change object content). This is of relatively important significance in some languages, so it is preserved

    - Type: Corresponding receiver struct Identity


- Params: Dependency array of types associated with input parameters (see [Dependency] below). If it is an anonymous parameter, ParamName is replaced by ParamTypeName


- Results: Dependency array of types associated with output parameters, {ResultName}:{Result Type Identity}. If it is an anonymous parameter, ParamName is replaced by ParamTypeName


- Content: Complete function content, including function signature + `\n` + function implementation code


- FunctionCalls: Array of other functions called within the current function. Arranged in the order they appear in the code (and deduplicated). Elements are corresponding AST node Identities


- MethodCalls: Array of methods called within the current function, arranged in the order they appear in the code (and deduplicated). Same rules as [FunctionCalls].


- Types: Types referenced within the current function, such as TypeX in `var x TypeX`


- Vars: Global variables referenced within the current function, including variables and constants


###### Dependency

Represents a dependency relationship, containing the dependent node Id, dependency location information, etc., to facilitate accurate identification by LLM


```
{
    "ModPath": "github.com/cloudwego/localsession",
    "PkgPath": "github.com/cloudwego/localsession",
    "Name": "transmitSessionIdentity",
    "File": "manager.go",
    "Line": 140,
    "StartOffset": 3547,
    "EndOffset": 3564
}
```

- ModPath: Module path, see [Identity] introduction


- PkgPath: Package path, see [Identity] introduction


- Name: Struct name


- File: Code file where the dependency point (not the dependent node) token is located


- Line: Code line where the dependency point (not the dependent node) token is located


- StartOffset: Offset of the starting position of the dependency point (not the dependent node) token relative to the code file


- EndOffset: Offset of the ending position of the dependency point (not the dependent node) token relative to the code file


##### Type

Type definition, [NodeType] is TYPE, including type definitions in specific languages such as structs, enums, interfaces, type aliases, etc.


```json
{
    "Exported": true,
    "TypeKind": "interface",
    "ModPath": "github.com/cloudwego/localsession",
    "PkgPath": "github.com/cloudwego/localsession",
    "Name": "Session",
    "File": "session.go",
    "Line": 25,
    "StartOffset": 725,
    "EndOffset": 1027,
    "Content": "// Session represents a local storage for one session\ntype Session interface {\n\t// IsValid tells if the session is valid at present\n\tIsValid() bool\n\n\t// Get returns value for specific key\n\tGet(key interface{}) interface{}\n\n\t// WithValue sets value for specific key，and return newly effective session\n\tWithValue(key interface{}, val interface{}) Session\n}",
    "InlineStruct": [
        {} // dependency
    ],
    "Methods": {
        "Get": {
            "ModPath": "github.com/cloudwego/localsession",
            "PkgPath": "github.com/cloudwego/localsession",
            "Name": "Session.Get"
        },
        "IsValid": {
            "ModPath": "github.com/cloudwego/localsession",
            "PkgPath": "github.com/cloudwego/localsession",
            "Name": "Session.IsValid"
        },
        "WithValue": {
            "ModPath": "github.com/cloudwego/localsession",
            "PkgPath": "github.com/cloudwego/localsession",
            "Name": "Session.WithValue"
        }
    },
    "Implements": []
}
```

- ModPath: Module path, see [Identity] introduction


- PkgPath: Package path, see [Identity] introduction


- Name: Struct name


- File: Filename where declared


- Line: Line number in the file where declared


- TypeKind: Type of kind -- No unified constraints here, defined by specific languages


- Exported: Whether visible/exported outside the package


- Content: Specific struct definition, including type signature + `\n` + type specific fields


- SubStructs: Dependency of sub-struct types referenced non-nested in fields (excluding go primitive types). The map key is the field name, and the value is the corresponding type AST node Identity


- InlineStructs: Dependency of sub-struct types referenced nested in fields (excluding go primitive types). The map key is the field name, and the value is the corresponding type AST node Identity

    - Reason: In some languages like Golang, methods of nested sub-structs are inherited by the parent struct, so they are distinguished from general sub-structs to facilitate tracing all methods owned by the type


- Methods: All method Identities corresponding to the struct. The key is the method name, and the value is the function Identity.

    - Note: This should not include methods of InlineStruct


- Implements: Which interfaces this type implements Identity


##### Var

Global variables, including variables and constants, **but must be global**


```rust
{
    "IsExported": false,
    "IsConst": false,
    "IsPointer": false,
    "ModPath": "github.com/cloudwego/localsession",
    "PkgPath": "github.com/cloudwego/localsession",
    "Name": "defaultShardCap",
    "File": "manager.go",
    "Line": 53,
    "StartOffset": 1501,
    "EndOffset": 1521,
    "Type": {
        "ModPath": "",
        "PkgPath": "",
        "Name": "int"
    },
    "Content": "var defaultShardCap int = 10"
}
```

- ModPath: Module path, see [Identity] introduction


- PkgPath: Package path, see [Identity] introduction


- Name: Variable name


- File: Filename where declared


- Line: Line number in the file where declared


- IsExported: Whether exported


- IsConst: Whether it is a constant


- Type: Identity corresponding to its type (excluding go primitive types). Go built-in types can only have name (e.g., string, uint)


- Content: Definition code, such as `var A int = 1 `

- Dependencies: Other nodes depended on in complex variable declaration bodies, such as 
```go
var x = getx(y db.Data) int {
    return y + model.Var2
}
```
中的 `db.Data` 和 `model.Var2`

- Groups: Group definitions, such as `const( A=1, B=2, C=3)` in Go, Groups would be `[C=3, B=2]` (assuming A is the variable itself)


### Graph

The dependency topology graph of all AST Nodes in the repository. Formatted as Identity => Node mapping, where each Node contains dependency relationships with other nodes.


```json
{
    "github.com/cloudwego/localsession?github.com/cloudwego/localsession#checkEnvOptions": {},
    "github.com/bytedance/gopkg@v0.0.0-20230728082804-614d0af6619b?github.com/bytedance/gopkg/cloud/metainfo#CountPersistentValues": {}
}
```

Where the key is obtained through the [Identity complete string] format


#### Node

A node represents an independent syntax unit, typically including code, location information, and dependency relationships


```go
{
    "ModPath": "github.com/cloudwego/localsession",
    "PkgPath": "github.com/cloudwego/localsession",
    "Name": "checkEnvOptions",
    "Type": "FUNC",
    "Dependencies": [
        {
            "Kind": "Dependency",
            "ModPath": "github.com/cloudwego/localsession",
            "PkgPath": "github.com/cloudwego/localsession",
            "Name": "SESSION_CONFIG_KEY",
            "Line": 1
        }
    ],
    "References": [
        {
            "Kind": "Reference",
            "ModPath": "github.com/cloudwego/localsession",
            "PkgPath": "github.com/cloudwego/localsession",
            "Name": "InitDefaultManager",
            "Line": 3
        }
    ],
    "Dependencies": [],
    "References": [],
    "Implements": [],
    "Inherits": [],
    "Groups": []
}
```

- ModPath: Target node module path, see [Identity] introduction


- PkgPath: Target node package path, see [Identity] introduction


- Name: Target node variable name


- Type: Target node type, see [NodeType] introduction


- Dependencies: Other nodes that this node depends on, each element is a Relation object


- References: Other nodes that depend on this node, each element is a Relation object


##### NodeType 

Includes three types:


```
// Node Type
type NodeType int

const (
    UNKNOWN NodeType = iota
    // top Function、 methods
    FUNC
    // Struct、TypeAlias、Enum...
    TYPE
    // Global Varable or Global Const
    VAR
)
```

- FUNC: Functions, including methods, top-level functions


- TYPE: Type definitions, including general type definitions such as structs, type aliases, interfaces


- VAR: Global variables or constants (excluding local variables, as we can collect local variables into FUNC or TYPE definitions)


#### Relation

Used to store relationships between two nodes. Example:
```
{
    "Kind": "Dependency",
    "ModPath": "github.com/cloudwego/localsession",
    "PkgPath": "github.com/cloudwego/localsession",
    "Name": "SESSION_CONFIG_KEY",
    "Line": 1,
    "Desc": "",
    "Codes": ""
}
```

- Kind: Relationship type, currently including:
  - Dependency: Dependency relationships, such as function calls, variable references, etc.

  - Implement: Implementation relationships, such as interface method implementations, etc.

  - Inherit: Inheritance relationships, such as struct fields, etc.

  - Group: Group definitions, such as `const( A=1; B=2; C=3)` in Go


- ModPath: Module path, see [Identity] introduction


- PkgPath: Package path, see [Identity] introduction


- Name: Variable name

- Line: The relative line number (starting from 0) where the relationship occurs in the main node's code


## Complete JSON Examples

-  https://github.com/cloudwego/localsession

    - Command: `git clone https://github.com/cloudwego/localsession.git && abcoder parse go ./localsession`

    - Output [localsession.json](../testdata/asts/localsession.json)


- https://github.com/cloudwego/metainfo

    - Command `git clone https://github.com/cloudwego/metainfo.git && abcoder parse rust ./metainfo -load-external-symbol`

    - Output [metainfo.json](../testdata/asts/metainfo.json)


# Extending Other Language Parsers

Currently, ABCoder/src/lang already supports third-party language parsing through LSP, but due to the lack of unified specifications for various language features (mainly function signatures and Import) in LSP, some interfaces need to be extended and implemented for adaptation. See [ABCoder-Language Plugin Development Specification](parser-en.md)