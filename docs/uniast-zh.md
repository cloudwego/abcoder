# Universal Abstract-Syntax-Tree Specification (v0.1.0)

Universal Abstract-Syntax-Tree 是 ABCoder 建立的一种LLM亲和、语言无关的代码上下文数据结构，表示某个仓库代码的统一抽象语法树。收集了语言实体（函数、类型、常（变）量）的 定义 及其 相互依赖关系，用于后续的 AI 理解、coding-workflow 开发。


## Identity 节点唯一标识

为了保证精确查询和可扩展存储，约定 **ModPath+PkgPath+SymbolName** 为 AST Node 的全球唯一标识**。


```json
{
    "ModPath": "github.com/cloudwego/localsession",
    "PkgPath": "github.com/cloudwego/localsession/backup",
    "Name": "RecoverCtxOnDemands"
}
```

- ModPath: 一个完整的构建单元，ModPath内容为安装路径@版本号。该信息对于LLM并不需要，只是为了保证 Identity 的全球唯一性而保存。它在各个语言中对应不同概念：

	- <u>Golang</u>：对应 module，如 github.com/cloudwego/hertz@v0.1.0

	- Rust：对应 crate，如 [serde_json](https://crates.io/crates/serde_json)@v1.0.114

	为了方便人工debug，这里有个实现约定：

	- 仓库内部函数或类型（包括**本包和本仓库内**其它子包函数）节点的Module**通常**不带版本号

	- 外部函数（第三方repo引入的函数）的 Module **通常**带上版本号 

	- 判断一个 Module 是否为第三方依赖尽量通过 Module.Dir 是否为空来判断，这里不保证


- PkgPath：语言中一个独立的命名空间，PkgPath对应语言中一个包的导入路径

	- Golang：对应 package，如 github.com/cloudwego/hertz/pkg/app/server

	- Rust：对应 mod，如 [serde_json](https://crates.io/crates/serde_json)::[value](https://docs.rs/serde_json/1.0.114/serde_json/value/index.html)

	- 提示：这里应该尽量等同于代码文件中的 import (use) 路径，方便 LLM 理解


- Name：在包内的唯一符号名
  
	- **如果节点为method，应该以**`TypeName.MethodName`**来表示。
  
	- **此外，有些语言如**rust允许一个类型为不同的接口实现同名方法**（比如rust），因此为了避免冲突TypeName可进一步扩展为` InterfaceName<ImplTypeName>` 形式


- **字符串（Key）形式**

	- Full() 完整形式为 `ModPath?PkgPath#Name`

	- String() Format形式为 `PkgPath#Name`，一般通过该形式展示给 LLM 即可


- **每个 AST 都会带有 Identity，但是是以内嵌的形式到具体节点字段中（Name、ModPath、PkgPath 三个字段）**





## 结构体定义（Go）

- 代码详见 [Repository](/lang/uniast/ast.go) 定义


## JSON 定义

以下以 [cloudwego/localsession](https://github.com/cloudwego/localsession.git) 库解析为示例介绍


## Repository

一个仓库由 实体Modules 和 关系Graph 组成


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

- Identity: repo 的唯一名称。由于abcoder parser目前不获取仓库git信息，因此一般使用当前所处的绝对路径作为Identity


- Modules：包含的子模块，{ModPath} : {Module AST} 的字典，本仓库模块和外部依赖模块都可以出现在Modules中，但是需要通过ModulePath来区分。

	- 本仓库模块ModePath == Module.Name

	- 外部依赖模块ModePath== Modele.Name@version


- Graph：AST Node 的依赖拓扑图，见下文【Graph】


### Module

代码独立编译单元，对应Identity中的ModPath，内部包含各个包


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

- Name：模块名（不带版本号）


- Language：代码使用的语言--对于多语言的仓库，一个模块的语言必须是唯一的。但是一个仓库可以有不同语言的模块。


- Dir：**模块与仓库根的相对路径。注意：**

	- **只有本仓库内的模块需要设置且不能为空**

	- **第三方依赖必须为空（当前用于判断是否为第三方依赖）**


- Dependencies：模块构建的第三方依赖模块字典 {ModName}:{ModPath}


- Packages: 包含的子包，{PkgPath}:{Pacakge AST} 字典


- Files：模块文件信息，key为**相对repo的路径。**这里建议包括仓库所有文件，方便writer回写


#### File

文件信息，包括代码文件和非代码文件都会记录


```json
// manager.go
{
    "Path": "manager.go",
    "Imports": [],
    "Package": "github.com/cloudwego/localsession"
}
```

- Path: 文件**相对仓库根的路径**


- Imports:  import 代码，


##### Import

```json
{
    "Alias": "_",
    "Path": "\"unsafe\""
}
```

- Path：导入路径主要用于writer写入代码，具体内容根据各个语言情况而定

	- rust 中为 `use xx::yy;`

	- Golang 中为 `"github.com/cloudwego/abcoder"`


- Alias：导入别名，可为空


#### Package

一个代码命名空间，对应Identity.PkgPath，内部包含各个AST Node实体


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

- PkgPath：模块路径，见【Identity】介绍


- IsMain: 是否是二进制包


- IsTest: 是否是测试包


- Functions：包含函数AST，  {FuncName}:{Function AST} 的字典


- Types：包含类型AST，{TypeName}:{Type AST}的字典


- Vars：包含全局变量/常量， {VarName}:{Variant AST} 的字典


##### Function

函数类型的AST Node实体，对应【NodeType】为 FUNC，包括函数、方法、接口函数


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
    "Content": "// BindSession binds the session with current goroutine\nfunc (self *SessionManager) BindSession(Identity SessionIdentity, s Session) {\n\tshard := self.shards[uint64(Identity)%uint64(self.opts.ShardNumber)]\n\n\tshard.Store(Identity, s)\n\n\tif self.opts.EnableImplicitlyTransmitAsync {\n\t\ttransmitSessionIdentity(Identity)\n\t}\n}",
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
    ]
}
```

- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name：函数名称

	- 如果函数为method，**应该**以 {TypeName}.{Methodname}来表示


- File：所在的文件名


- Line：**起始位置文件的行号**


- StartOffset：代码起始位置**相对文件头的字节偏移量** 


- EndOffset:  代码结束位置**相对文件头的字节偏移量**


- Exported：是否包外可见导出


- IsMethod：是否是一个方法


- IsInterfaceMethod：是否是接口的方法--这里abcoder parse收集InterfaceMethod为了方便LLM理解，但是实际上write中并不会认为其是一个语言实体


- Receiver：如果是方法的话，会有的receiver结构体。

	- IsPointer：是否是指针接受者（可改变对象内容）。这个在某些语言中有比较重要意义，因此保留

	- Type：对应的receiver结构体Identity


- Params：入参中关联的类型的Dependency数组（见下文【Dependency】），如果是匿名信参数ParamName由ParamTypeName替代


- Results：出参中关联的类型Dependency数组， {ResultName}:{Result Type Identity}，如果是匿名信参数ParamName由ParamTypeName替代


- Content：函数完整内容，包括函数签名+`\n`+函数实现代码


- FunctionCalls：当前函数中调用的其他函数Dependency数组。按依赖在代码中出现的次序排列（并去重）。元素为对应的 AST 节点 Identity


- MethodCalls：当前函数中调用的方法Dependency数组，按依赖在代码中出现的次序排列（并去重）。规则同【FunctionCalls】。


- Types： 当前函数内引用的类型，如 `var x TypeX`中的TypeX


- Vars：当前函数内引用的全局量，包括变量和常量


###### Dependency

表示一个依赖关系，包含依赖节点Id、依赖产生位置等信息，方便LLM准确识别


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

- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name： 结构体名称


- File：依赖点（不是被依赖节点）token所处的代码文件


- Line：依赖点（不是被依赖节点）token所处的代码行


- StartOffset：依赖点（不是被依赖节点）token起始位置相对代码文件的偏移


- EndOffset：依赖点（不是被依赖节点）token结束位置相对代码文件的偏移


##### Type

类型定义，【NodeType】为 TYPE，包括具体语言中的类型定义，如 结构体、枚举、接口、类型别名等


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
    }
}
```

- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name： 结构体名称


- File：声明所在的文件名


- Line：声明所在文件的行号


- TypeKind：类型的种类 Kind -- 这里不做统一约束，由具体语言定义


- Exported：是否包外可见导出


- Content：具体结构体定义，包括类型签名+`\n`+类型具体字段


- SubStructs：字段中非嵌套引用的子结构体类型**Dependency**（不包括 go 原始类型），map key为字段名，val为对应类型 AST 节点 Identity


- InlineStructs: 字段中嵌套引用的子结构体类型**Dependency**（不包括 go 原始类型），map key为字段名，val对应类型 AST 节点 Identity

	- 原因：在某些语言如 Golang 中嵌套子结构体的 methods 会被继承到父结构体中，因此和一般子结构体区分开，方便回溯该类型拥有的所有 method


- Methods：结构体对应的全部方法**Identity**，key为方法名，val为函数Identity。

	- 注意这里不应该包括 InlineStruct 的 methods


- Implements：该类型实现了哪些接口**Identity**


##### Var

全局量，包括变量和常量，**但是必须是全局**


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

- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name： 变量名称


- File：声明所在的文件名


- Line：声明所在文件的行号


- IsExported：是否导出


- IsConst：是否为常量


- Type：其类型对应的Identity（不包括 go 原始类型），go内置类型可以只有name（如 string, uint）


- Content：定义代码，如 `var A int = 1 `


### Graph

整个仓库的AST Node依赖拓扑图。形式为 Identity => Node 的映射，其中每个 Node 包含对其它节点的依赖关系。基于该拓扑图，可以实现**任意节点上下文的递归获取**。


```json
{
    "github.com/cloudwego/localsession?github.com/cloudwego/localsession#checkEnvOptions": {},
    "github.com/bytedance/gopkg@v0.0.0-20230728082804-614d0af6619b?github.com/bytedance/gopkg/cloud/metainfo#CountPersistentValues": {}
}
```

其中 key 通过 【Identity 的完整字符串】形式得到


#### Node

一个node表示一个独立的语法单元，通常包括代码、位置信息和依赖关系


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
    ]
}
```

- ModPath: 目标节点模块路径，见【Identity】介绍


- PkgPath：目标节点包路径，见【Identity】介绍


- Name： 目标节点变量名称


- Type： 目标节点类型，见【NodeType】介绍


- Dependencies：该节点依赖的其他节点，每个元素对象为Relation


- References：依赖该节点的其他节点，每个元素对象为Relation


##### NodeType 

包括三种类型：


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

- FUNC：函数，包括方法、顶层函数


- TYPE：类型定义，包括 struct、类型别名、接口等通用的类型定义


- VAR：全局变量或常量（不包括局部变量，因为我们局部变量可以收集到FUNC或TYPE定义中）


#### Relation

用于存储两个节点之间的关系。


```
{
    "Kind": "Dependency",
    "ModPath": "github.com/cloudwego/localsession",
    "PkgPath": "github.com/cloudwego/localsession",
    "Name": "getPproLabel",
    "Line": 1
}
```

- Kind： 关系类型，目前包括 DEPENDENCY 和 REFERENCE


- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name： 变量名称


- Line: 产生关系的位置在主节点代码的行号（即函数体内的行号）


## 完整JSON示例

-  https://github.com/cloudwego/localsession

	- 命令：` git clone https://github.com/cloudwego/localsession.git && abcoder parse go ./localsession -load-external-symbol`

	- 输出  


- https://github.com/cloudwego/metainfo

	- 命令`git clone https://github.com/cloudwego/metainfo.git && abcoder parse rust ./metainfo -load-external-symbol`

	- 输出 


## 扩展其它语言 Parser

当前ABCoder/src/lang 已经支持通过LSP来进行第三方语言解析，但是由于LSP对各个语言特性（主要是函数签名和Import）没有统一规范，因此需要扩展实现一些接口才能适配。详见 [ABCoder-Language Plugin 开发规范](https://bytedance.sg.larkoffice.com/docx/QNZwdYBPjoUUKtxD3XMlstlpglh)
Universal Abstract-Syntax-Tree 是 ABCoder 建立的一种LLM亲和、语言无关的代码上下文数据结构，表示某个仓库代码的统一抽象语法树。收集了语言实体（函数、类型、常（变）量）的 定义 及其 相互依赖关系，用于后续的 AI 理解、coding-workflow 开发。


## Identity 节点唯一标识

为了保证精确查询和可扩展存储，约定 ModPath**+**PkgPath**+**SymbolName 为 AST Node 的**全球唯一标识**。


```json
{
    "ModPath": "github.com/cloudwego/localsession",
    "PkgPath": "github.com/cloudwego/localsession/backup",
    "Name": "RecoverCtxOnDemands"
}
```

- ModPath: 一个完整的构建单元，ModPath内容为安装路径@版本号。该信息对于LLM并不需要，只是为了保证 Identity 的全球唯一性而保存。它在各个语言中对应不同概念：

	- <u>Golang</u>：对应 module，如 github.com/cloudwego/hertz@v0.1.0

	- Rust：对应 crate，如 [serde_json](https://crates.io/crates/serde_json)@v1.0.114

	为了方便人工debug，这里有个实现约定：

	- 仓库内部函数或类型（包括**本包和本仓库内**其它子包函数）节点的Module**通常**不带版本号

	- 外部函数（第三方repo引入的函数）的 Module **通常**带上版本号 

	- 判断一个 Module 是否为第三方依赖尽量通过 Module.Dir 是否为空来判断，这里不保证


- PkgPath：语言中一个独立的命名空间，PkgPath对应语言中一个包的import path**：**

	- Golang：对应 package，如 github.com/cloudwego/hertz/pkg/app/server

	- Rust：对应 mod，如 [serde_json](https://crates.io/crates/serde_json)::[value](https://docs.rs/serde_json/1.0.114/serde_json/value/index.html)

	- 提示：这里应该尽量等同于代码文件中的 import (use) 路径，方便 LLM 理解


- Name：在包内的唯一符号名

	- 如果节点类型

	- **如果节点为method，应该以**`TypeName.MethodName`**来表示。**此外，有些语言如**rust允许一个类型为不同的接口实现同名方法**（比如rust），因此为了避免冲突TypeName可进一步扩展为` InterfaceName<ImplTypeName>` 形式


- **字符串（Key）形式**

	- Full() 完整形式为 `ModPath?PkgPath#Name`

	- String() Format形式为 PkgPath#Name，一般通过该形式展示给 LLM 即可


- **每个 AST 都会带有 Identity，但是是以内嵌的形式到具体节点字段中（Name、ModPath、PkgPath 三个字段）**





## 结构体定义（Go）

- 代码详见 abcoder/lang/uniast#Repository


## JSON 定义

以下以 `[cloudwego/localsession](https://github.com/cloudwego/localsession.git)` 库解析为示例介绍


## Repository

一个仓库由 实体Modules 和 关系Graph 组成


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

- Identity: repo 的唯一名称。由于abcoder parser目前不获取仓库git信息，因此一般使用当前所处的绝对路径作为Identity


- Modules：包含的子模块，{ModPath} : {Module AST} 的字典，本仓库模块和外部依赖模块都可以出现在Modules中，但是需要通过ModulePath来区分。

	- 本仓库模块ModePath == Module.Name

	- 外部依赖模块ModePath== Modele.Name@version


- Graph：AST Node 的依赖拓扑图，见下文【Graph】


### Module

代码独立编译单元，对应Identity中的ModPath，内部包含各个包


```json
{
    "Name": "github.com/cloudwego/localsession"
    "Language": "go",
    "Version": "",
    "Name": "github.com/cloudwego/localsession",
    "Dir": ".",
    "Packages": {
        "github.com/cloudwego/localsession": {},
        "github.com/cloudwego/localsession/backup": {}
    }
    "Dependencies": {
        "github.com/bytedance/gopkg": "github.com/bytedance/gopkg@v0.0.0-20230728082804-614d0af6619b"
    },
    "Files": {
        ".github/ISSUE_TEMPLATE/bug_report.md": {},
        "backup/metainfo.go": {}
    }
}
```

- Name：模块名（不带版本号）


- Language：代码使用的语言--对于多语言的仓库，一个模块的语言必须是唯一的。但是一个仓库可以有不同语言的模块。


- Dir：**模块与仓库根的相对路径。注意：**

	- **只有本仓库内的模块需要设置且不能为空**

	- **第三方依赖必须为空（当前用于判断是否为第三方依赖）**


- Dependencies：模块构建的第三方依赖模块字典 {ModName}:{ModPath}


- Packages: 包含的子包，{PkgPath}:{Pacakge AST} 字典


- Files：模块文件信息，key为**相对repo的路径。**这里建议包括仓库所有文件，方便writer回写


#### File

文件信息，包括代码文件和非代码文件都会记录


```json
// manager.go
{
    "Path": "manager.go",
    "Imports": [],
    "Package": "github.com/cloudwego/localsession"
}
```

- Path: 文件**相对仓库根的路径**


- Imports:  import 代码，


##### Import

```json
{
    "Alias": "_",
    "Path": "\"unsafe\""
}
```

- Path：导入路径主要用于writer写入代码，具体内容根据各个语言情况而定

	- rust 中为 `use xx::yy;`

	- Golang 中为 `"github.com/cloudwego/abcoder"`


- Alias：导入别名，可为空


#### Package

一个代码命名空间，对应Identity.PkgPath，内部包含各个AST Node实体


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

- PkgPath：模块路径，见【Identity】介绍


- IsMain: 是否是二进制包


- IsTest: 是否是测试包


- Functions：包含函数AST，  {FuncName}:{Function AST} 的字典


- Types：包含类型AST，{TypeName}:{Type AST}的字典


- Vars：包含全局变量/常量， {VarName}:{Variant AST} 的字典


##### Function

函数类型的AST Node实体，对应【NodeType】为 FUNC，包括函数、方法、接口函数


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
    "Content": "// BindSession binds the session with current goroutine\nfunc (self *SessionManager) BindSession(Identity SessionIdentity, s Session) {\n\tshard := self.shards[uint64(Identity)%uint64(self.opts.ShardNumber)]\n\n\tshard.Store(Identity, s)\n\n\tif self.opts.EnableImplicitlyTransmitAsync {\n\t\ttransmitSessionIdentity(Identity)\n\t}\n}",
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
    ]
}
```

- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name：函数名称

	- 如果函数为method，**应该**以 {TypeName}.{Methodname}来表示


- File：所在的文件名


- Line：**起始位置文件的行号**


- StartOffset：代码起始位置**相对文件头的字节偏移量** 


- EndOffset:  代码结束位置**相对文件头的字节偏移量**


- Exported：是否包外可见导出


- IsMethod：是否是一个方法


- IsInterfaceMethod：是否是接口的方法--这里abcoder parse收集InterfaceMethod为了方便LLM理解，但是实际上write中并不会认为其是一个语言实体


- Receiver：如果是方法的话，会有的receiver结构体。

	- IsPointer：是否是指针接受者（可改变对象内容）。这个在某些语言中有比较重要意义，因此保留

	- Type：对应的receiver结构体Identity


- Params：入参中关联的类型的Dependency数组（见下文【Dependency】），如果是匿名信参数ParamName由ParamTypeName替代


- Results：出参中关联的类型Dependency数组， {ResultName}:{Result Type Identity}，如果是匿名信参数ParamName由ParamTypeName替代


- Content：函数完整内容，包括函数签名+`\n`+函数实现代码


- FunctionCalls：当前函数中调用的其他函数Dependency数组。按依赖在代码中出现的次序排列（并去重）。元素为对应的 AST 节点 Identity


- MethodCalls：当前函数中调用的方法Dependency数组，按依赖在代码中出现的次序排列（并去重）。规则同【FunctionCalls】。


- Types： 当前函数内引用的类型，如 `var x TypeX`中的TypeX


- Vars：当前函数内引用的全局量，包括变量和常量


###### Dependency

表示一个依赖关系，包含依赖节点Id、依赖产生位置等信息，方便LLM准确识别


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

- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name： 结构体名称


- File：依赖点（不是被依赖节点）token所处的代码文件


- Line：依赖点（不是被依赖节点）token所处的代码行


- StartOffset：依赖点（不是被依赖节点）token起始位置相对代码文件的偏移


- EndOffset：依赖点（不是被依赖节点）token结束位置相对代码文件的偏移


##### Type

类型定义，【NodeType】为 TYPE，包括具体语言中的类型定义，如 结构体、枚举、接口、类型别名等


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
    }
}
```

- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name： 结构体名称


- File：声明所在的文件名


- Line：声明所在文件的行号


- TypeKind：类型的种类 Kind -- 这里不做统一约束，由具体语言定义


- Exported：是否包外可见导出


- Content：具体结构体定义，包括类型签名+`\n`+类型具体字段


- SubStructs：字段中非嵌套引用的子结构体类型**Dependency**（不包括 go 原始类型），map key为字段名，val为对应类型 AST 节点 Identity


- InlineStructs: 字段中嵌套引用的子结构体类型**Dependency**（不包括 go 原始类型），map key为字段名，val对应类型 AST 节点 Identity

	- 原因：在某些语言如 Golang 中嵌套子结构体的 methods 会被继承到父结构体中，因此和一般子结构体区分开，方便回溯该类型拥有的所有 method


- Methods：结构体对应的全部方法**Identity**，key为方法名，val为函数Identity。

	- 注意这里不应该包括 InlineStruct 的 methods


- Implements：该类型实现了哪些接口**Identity**


##### Var

全局量，包括变量和常量，**但是必须是全局**


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

- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name： 变量名称


- File：声明所在的文件名


- Line：声明所在文件的行号


- IsExported：是否导出


- IsConst：是否为常量


- Type：其类型对应的Identity（不包括 go 原始类型），go内置类型可以只有name（如 string, uint）


- Content：定义代码，如 `var A int = 1 `


### Graph

整个仓库的AST Node依赖拓扑图。形式为 Identity => Node 的映射，其中每个 Node 包含对其它节点的依赖关系。基于该拓扑图，可以实现**任意节点上下文的递归获取**。


```json
{
    "github.com/cloudwego/localsession?github.com/cloudwego/localsession#checkEnvOptions": {},
    "github.com/bytedance/gopkg@v0.0.0-20230728082804-614d0af6619b?github.com/bytedance/gopkg/cloud/metainfo#CountPersistentValues": {}
}
```

其中 key 通过 【Identity 的完整字符串】形式得到


#### Node

一个node表示一个独立的语法单元，通常包括代码、位置信息和依赖关系


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
    ]
}
```

- ModPath: 目标节点模块路径，见【Identity】介绍


- PkgPath：目标节点包路径，见【Identity】介绍


- Name： 目标节点变量名称


- Type： 目标节点类型，见【NodeType】介绍


- Dependencies：该节点依赖的其他节点，每个元素对象为Relation


- References：依赖该节点的其他节点，每个元素对象为Relation


##### NodeType 

包括三种类型：


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

- FUNC：函数，包括方法、顶层函数


- TYPE：类型定义，包括 struct、类型别名、接口等通用的类型定义


- VAR：全局变量或常量（不包括局部变量，因为我们局部变量可以收集到FUNC或TYPE定义中）


#### Relation

用于存储两个节点之间的关系。


```
{
    "Kind": "Dependency",
    "ModPath": "github.com/cloudwego/localsession",
    "PkgPath": "github.com/cloudwego/localsession",
    "Name": "getPproLabel",
    "Line": 1
}
```

- Kind： 关系类型，目前包括 DEPENDENCY 和 REFERENCE


- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name： 变量名称


- Line: 产生关系的位置在主节点代码的行号（即函数体内的行号）


## 完整JSON示例

-  https://github.com/cloudwego/localsession

	- 命令：` git clone https://github.com/cloudwego/localsession.git && abcoder parse go ./localsession -load-external-symbol`

	- 输出  


- https://github.com/cloudwego/metainfo

	- 命令`git clone https://github.com/cloudwego/metainfo.git && abcoder parse rust ./metainfo -load-external-symbol`

	- 输出 


## 扩展其它语言 Parser

当前ABCoder/src/lang 已经支持通过LSP来进行第三方语言解析，但是由于LSP对各个语言特性（主要是函数签名和Import）没有统一规范，因此需要扩展实现一些接口才能适配。详见 [ABCoder-Language Plugin 开发规范](https://bytedance.sg.larkoffice.com/docx/QNZwdYBPjoUUKtxD3XMlstlpglh)
Universal Abstract-Syntax-Tree 是 ABCoder 建立的一种LLM亲和、语言无关的代码上下文数据结构，表示某个仓库代码的统一抽象语法树。收集了语言实体（函数、类型、常（变）量）的 定义 及其 相互依赖关系，用于后续的 AI 理解、coding-workflow 开发。


## Identity 节点唯一标识

为了保证精确查询和可扩展存储，约定 ModPath**+**PkgPath**+**SymbolName 为 AST Node 的**全球唯一标识**。


```json
{
    "ModPath": "github.com/cloudwego/localsession",
    "PkgPath": "github.com/cloudwego/localsession/backup",
    "Name": "RecoverCtxOnDemands"
}
```

- ModPath: 一个完整的构建单元，ModPath内容为安装路径@版本号。该信息对于LLM并不需要，只是为了保证 Identity 的全球唯一性而保存。它在各个语言中对应不同概念：

	- <u>Golang</u>：对应 module，如 github.com/cloudwego/hertz@v0.1.0

	- Rust：对应 crate，如 [serde_json](https://crates.io/crates/serde_json)@v1.0.114

	为了方便人工debug，这里有个实现约定：

	- 仓库内部函数或类型（包括**本包和本仓库内**其它子包函数）节点的Module**通常**不带版本号

	- 外部函数（第三方repo引入的函数）的 Module **通常**带上版本号 

	- 判断一个 Module 是否为第三方依赖尽量通过 Module.Dir 是否为空来判断，这里不保证


- PkgPath：语言中一个独立的命名空间，PkgPath对应语言中一个包的import path**：**

	- Golang：对应 package，如 github.com/cloudwego/hertz/pkg/app/server

	- Rust：对应 mod，如 [serde_json](https://crates.io/crates/serde_json)::[value](https://docs.rs/serde_json/1.0.114/serde_json/value/index.html)

	- 提示：这里应该尽量等同于代码文件中的 import (use) 路径，方便 LLM 理解


- Name：在包内的唯一符号名

	- 如果节点类型

	- **如果节点为method，应该以**`TypeName.MethodName`**来表示。**此外，有些语言如**rust允许一个类型为不同的接口实现同名方法**（比如rust），因此为了避免冲突TypeName可进一步扩展为` InterfaceName<ImplTypeName>` 形式


- **字符串（Key）形式**

	- Full() 完整形式为 `ModPath?PkgPath#Name`

	- String() Format形式为 PkgPath#Name，一般通过该形式展示给 LLM 即可


- **每个 AST 都会带有 Identity，但是是以内嵌的形式到具体节点字段中（Name、ModPath、PkgPath 三个字段）**





## 结构体定义（Go）

- 代码详见 abcoder/lang/uniast#Repository


## JSON 定义

以下以 `[cloudwego/localsession](https://github.com/cloudwego/localsession.git)` 库解析为示例介绍


## Repository

一个仓库由 实体Modules 和 关系Graph 组成


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

- Identity: repo 的唯一名称。由于abcoder parser目前不获取仓库git信息，因此一般使用当前所处的绝对路径作为Identity


- Modules：包含的子模块，{ModPath} : {Module AST} 的字典，本仓库模块和外部依赖模块都可以出现在Modules中，但是需要通过ModulePath来区分。

	- 本仓库模块ModePath == Module.Name

	- 外部依赖模块ModePath== Modele.Name@version


- Graph：AST Node 的依赖拓扑图，见下文【Graph】


### Module

代码独立编译单元，对应Identity中的ModPath，内部包含各个包


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

- Name：模块名（不带版本号）


- Language：代码使用的语言--对于多语言的仓库，一个模块的语言必须是唯一的。但是一个仓库可以有不同语言的模块。


- Dir：**模块与仓库根的相对路径。注意：**

	- **只有本仓库内的模块需要设置且不能为空**

	- **第三方依赖必须为空（当前用于判断是否为第三方依赖）**


- Dependencies：模块构建的第三方依赖模块字典 {ModName}:{ModPath}


- Packages: 包含的子包，{PkgPath}:{Pacakge AST} 字典


- Files：模块文件信息，key为**相对repo的路径。**这里建议包括仓库所有文件，方便writer回写


#### File

文件信息，包括代码文件和非代码文件都会记录


```json
// manager.go
{
    "Path": "manager.go",
    "Imports": [],
    "Package": "github.com/cloudwego/localsession"
}
```

- Path: 文件**相对仓库根的路径**


- Imports:  import 代码，


##### Import

```json
{
    "Alias": "_",
    "Path": "\"unsafe\""
}
```

- Path：导入路径主要用于writer写入代码，具体内容根据各个语言情况而定

	- rust 中为 `use xx::yy;`

	- Golang 中为 `"github.com/cloudwego/abcoder"`


- Alias：导入别名，可为空


#### Package

一个代码命名空间，对应Identity.PkgPath，内部包含各个AST Node实体


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

- PkgPath：模块路径，见【Identity】介绍


- IsMain: 是否是二进制包


- IsTest: 是否是测试包


- Functions：包含函数AST，  {FuncName}:{Function AST} 的字典


- Types：包含类型AST，{TypeName}:{Type AST}的字典


- Vars：包含全局变量/常量， {VarName}:{Variant AST} 的字典


##### Function

函数类型的AST Node实体，对应【NodeType】为 FUNC，包括函数、方法、接口函数


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
    "Content": "// BindSession binds the session with current goroutine\nfunc (self *SessionManager) BindSession(Identity SessionIdentity, s Session) {\n\tshard := self.shards[uint64(Identity)%uint64(self.opts.ShardNumber)]\n\n\tshard.Store(Identity, s)\n\n\tif self.opts.EnableImplicitlyTransmitAsync {\n\t\ttransmitSessionIdentity(Identity)\n\t}\n}",
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
    ]
}
```

- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name：函数名称

	- 如果函数为method，**应该**以 {TypeName}.{Methodname}来表示


- File：所在的文件名


- Line：**起始位置文件的行号**


- StartOffset：代码起始位置**相对文件头的字节偏移量** 


- EndOffset:  代码结束位置**相对文件头的字节偏移量**


- Exported：是否包外可见导出


- IsMethod：是否是一个方法


- IsInterfaceMethod：是否是接口的方法--这里abcoder parse收集InterfaceMethod为了方便LLM理解，但是实际上write中并不会认为其是一个语言实体


- Receiver：如果是方法的话，会有的receiver结构体。

	- IsPointer：是否是指针接受者（可改变对象内容）。这个在某些语言中有比较重要意义，因此保留

	- Type：对应的receiver结构体Identity


- Params：入参中关联的类型的Dependency数组（见下文【Dependency】），如果是匿名信参数ParamName由ParamTypeName替代


- Results：出参中关联的类型Dependency数组， {ResultName}:{Result Type Identity}，如果是匿名信参数ParamName由ParamTypeName替代


- Content：函数完整内容，包括函数签名+`\n`+函数实现代码


- FunctionCalls：当前函数中调用的其他函数Dependency数组。按依赖在代码中出现的次序排列（并去重）。元素为对应的 AST 节点 Identity


- MethodCalls：当前函数中调用的方法Dependency数组，按依赖在代码中出现的次序排列（并去重）。规则同【FunctionCalls】。


- Types： 当前函数内引用的类型，如 `var x TypeX`中的TypeX


- Vars：当前函数内引用的全局量，包括变量和常量


###### Dependency

表示一个依赖关系，包含依赖节点Id、依赖产生位置等信息，方便LLM准确识别


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

- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name： 结构体名称


- File：依赖点（不是被依赖节点）token所处的代码文件


- Line：依赖点（不是被依赖节点）token所处的代码行


- StartOffset：依赖点（不是被依赖节点）token起始位置相对代码文件的偏移


- EndOffset：依赖点（不是被依赖节点）token结束位置相对代码文件的偏移


##### Type

类型定义，【NodeType】为 TYPE，包括具体语言中的类型定义，如 结构体、枚举、接口、类型别名等


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
    }
}
```

- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name： 结构体名称


- File：声明所在的文件名


- Line：声明所在文件的行号


- TypeKind：类型的种类 Kind -- 这里不做统一约束，由具体语言定义


- Exported：是否包外可见导出


- Content：具体结构体定义，包括类型签名+`\n`+类型具体字段


- SubStructs：字段中非嵌套引用的子结构体类型**Dependency**（不包括 go 原始类型），map key为字段名，val为对应类型 AST 节点 Identity


- InlineStructs: 字段中嵌套引用的子结构体类型**Dependency**（不包括 go 原始类型），map key为字段名，val对应类型 AST 节点 Identity

	- 原因：在某些语言如 Golang 中嵌套子结构体的 methods 会被继承到父结构体中，因此和一般子结构体区分开，方便回溯该类型拥有的所有 method


- Methods：结构体对应的全部方法**Identity**，key为方法名，val为函数Identity。

	- 注意这里不应该包括 InlineStruct 的 methods


- Implements：该类型实现了哪些接口**Identity**


##### Var

全局量，包括变量和常量，**但是必须是全局**


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

- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name： 变量名称


- File：声明所在的文件名


- Line：声明所在文件的行号


- IsExported：是否导出


- IsConst：是否为常量


- Type：其类型对应的Identity（不包括 go 原始类型），go内置类型可以只有name（如 string, uint）


- Content：定义代码，如 `var A int = 1 `


### Graph

整个仓库的AST Node依赖拓扑图。形式为 Identity => Node 的映射，其中每个 Node 包含对其它节点的依赖关系。基于该拓扑图，可以实现**任意节点上下文的递归获取**。


```json
{
    "github.com/cloudwego/localsession?github.com/cloudwego/localsession#checkEnvOptions": {},
    "github.com/bytedance/gopkg@v0.0.0-20230728082804-614d0af6619b?github.com/bytedance/gopkg/cloud/metainfo#CountPersistentValues": {}
}
```

其中 key 通过 【Identity 的完整字符串】形式得到


#### Node

一个node表示一个独立的语法单元，通常包括代码、位置信息和依赖关系


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
    ]
}
```

- ModPath: 目标节点模块路径，见【Identity】介绍


- PkgPath：目标节点包路径，见【Identity】介绍


- Name： 目标节点变量名称


- Type： 目标节点类型，见【NodeType】介绍


- Dependencies：该节点依赖的其他节点，每个元素对象为Relation


- References：依赖该节点的其他节点，每个元素对象为Relation


##### NodeType 

包括三种类型：


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

- FUNC：函数，包括方法、顶层函数


- TYPE：类型定义，包括 struct、类型别名、接口等通用的类型定义


- VAR：全局变量或常量（不包括局部变量，因为我们局部变量可以收集到FUNC或TYPE定义中）


#### Relation

用于存储两个节点之间的关系。


```
{
    "Kind": "Dependency",
    "ModPath": "github.com/cloudwego/localsession",
    "PkgPath": "github.com/cloudwego/localsession",
    "Name": "getPproLabel",
    "Line": 1
}
```

- Kind： 关系类型，目前包括 DEPENDENCY 和 REFERENCE


- ModPath: 模块路径，见【Identity】介绍


- PkgPath：包路径，见【Identity】介绍


- Name： 变量名称


- Line: 产生关系的位置在主节点代码的行号（即函数体内的行号）


## 完整JSON示例

-  https://github.com/cloudwego/localsession

	- 命令：` git clone https://github.com/cloudwego/localsession.git && abcoder parse go ./localsession -load-external-symbol`

	- 输出 [localsession.json](localsession.json)


- https://github.com/cloudwego/metainfo

	- 命令`git clone https://github.com/cloudwego/metainfo.git && abcoder parse rust ./metainfo -load-external-symbol`

	- 输出 [metainfo.json](metainfo.json)


## 扩展其它语言 Parser

当前ABCoder/src/lang 已经支持通过LSP来进行第三方语言解析，但是由于LSP对各个语言特性（主要是函数签名和Import）没有统一规范，因此需要扩展实现一些接口才能适配。详见 [ABCoder-Language Plugin 开发规范](parser-zh.md)
