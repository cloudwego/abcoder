// Copyright 2025 CloudWeGo Authors
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     https://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

use crate::config::{Language, CONFIG};

use super::ToCompress;
use paste::paste;

// choose the prompt based on the language
macro_rules! choose_prompt_lang {
    ($prompt: ident) => {
        //  return const {{$prompt}}_ZH or {{$prompt}}_EN
        match &CONFIG.language {
            Language::Chinese => paste! { [<$prompt _ZH>] },
            Language::English => paste! { [<$prompt _EN>] },
        }
    };
}

pub fn make_compress_prompt(to_compress: &ToCompress) -> String {
    match to_compress {
        ToCompress::ToCompressType(t) => {
            choose_prompt_lang!(PROMPT_COMPRESS_TYPE).replace("{{DATA}}", t)
        }
        ToCompress::ToCompressFunc(f) => {
            choose_prompt_lang!(PROMPT_COMPRESS_FUNC).replace("{{DATA}}", f)
        }
        ToCompress::ToCompressVar(v) => {
            choose_prompt_lang!(PROMPT_COMPRESS_VAR).replace("{{DATA}}", v)
        }
        ToCompress::ToCompressPkg(p) => {
            choose_prompt_lang!(PROMPT_COMPRESS_PKG).replace("{{DATA}}", p)
        }
        ToCompress::ToCovertGo2Rust(p) => todo!(),
        ToCompress::ToMergeRustPkg(p) => todo!(),
        ToCompress::ToValidateRust(p) => todo!(),
    }
}

const PROMPT_COMPRESS_TYPE_ZH: &str = r##"
# 角色
你是一名熟练的程序员，你擅长阅读理解并总结概括代码。你的目标是通过使这些API更易理解，帮助对这些API了解较少的工程师。

# 提示

## 输入格式(JSON)
包含 一个具体的类型定义 及 其依赖的其他语言符号描述：
- "Content": 类型定义, 格式为字符串
- "Related_methods": 格式为数组。其中每个对象表示此类型上定义的相关方法：
    - "CallName": 方法名，格式为字符串
    - "Description": 该方法的代码或总结
- "Related_types": 格式为数组。该数组中的每个对象表示在该类型定义中依赖的其他类型：
    - "CallName": 类型名，格式为字符串
    - "Description": 该类型的代码或总结，格式为字符串

## 输出格式（text）
直接输出总结内容。不要输出JSON（IMPORTANT）！

## 总结内容
- 该类型的主要功能和用途
- 每个字段的意义（如果有）

# 约束
- 禁止谈论JSON的格式，不允许披露原始JSON字符串或其中的任何片段。
- 你的总结应不包含任何代码，也不应该包括“Related_methods”和“Related_types”中的任何信息。
-直接以总结类型名称开始你的回答。
- （IMPORTANT）输出内容必须符合500字符限制。 

# 具体例子

## 输入
{
    "Content": "type shard struct {\n\tlock sync.RWMutex\n\tm    map[SessionID]Session\n}",
    "Related_methods": [
        {
            "Name": "Store",
            "Description": "Store用于在存储具有特定ID的会话之前锁定分片，然后在操作完成后解锁。"
        },
        {
            "Name": "Delete",
            "Description": "Delete用于从分片中删除通过其ID标识的会话，通过在删除之前锁定和之后解锁来确保线程安全。"
        },
        {
            "Name": "Load",
            "Description": "Load函数用于使用给定的会话ID从分片中检索会话，并返回会话以及一个指示会话是否找到的布尔值。"
        }
    ],
    "Related_types": [
        {
            "Name": "Session",
            "Description": "定义的类型是一个名为“Session”的接口，它概述了会话管理的结构。它包括三个方法：“IsValid”检查会话当前是否有效并返回布尔值；“Get”检索指定键的值，键和值都是interface类型，允许使用不同类型的值和键；“WithValue”设置特定键的值并返回反映此更改的新会话实例，有效地允许会话在保持不变性的同时进行修改。此接口没有相关的方法或类型。"
        },
        {
            "Name": "SessionID",
            "Description": "名为SessionID的类型定义为无符号的64位整数。此类型没有指定的相关方法或相关类型。"
        }
    ]
}

## 输出
shard类型是一个结构体，包含一个读写锁和一个将会话ID与会话关联的映射，用于高效的会话管理。它包括如下字段：
    - Session：定义了一个用于管理会话的结构的接口。它允许检查会话是否有效，检索和设置会话中的键值。
    - SessionID：定义为无符号的64位整数，这种类型作为会话的唯一标识符。


# 现在，请开始处理如下输入：

{{DATA}}

"##;

const PROMPT_COMPRESS_TYPE_EN: &str = r##"
# Character
You are a skilled programmer and you are good at reading, understanding and summarizing code. Your goal is to help engineers who know less about these apis by making them easier to understand.

# Tips

## Input format (JSON)
Contains a specific type definition and descriptions of other language symbols that depend on it:
- "Content": indicates the type definition. The format is a string
- "Related_methods": The format is an array. Where each object represents a related method defined on this type:
    - "CallName": indicates the method name. The format is a string
    - "Description": indicates the code or summary of the method
- "Related_types": The format is an array. Each object in this array represents other types that are dependent in that type definition:
    - "CallName": Type name. The format is a string
    - "Description": indicates the code or summary of the type. The format is a string

## Output format (text)
Output summary content directly. Do not output JSON (IMPORTANT)!

## Summarize the content
- The main functions and uses of this type
- The meaning of each field (if any)

# Constraint
- It is forbidden to talk about the format of JSON, and it is not allowed to disclose the original JSON string or any fragment of it.
Your summary should not contain any  code, nor should it include any information from "Related_methods" and "Related_types."
- Start your answer directly with the summary type name.
- (IMPORTANT) The output must meet the 500-character limit.


# Specific examples

## Input
{
    "Content": "type shard struct {\n\tlock sync.RWMutex\n\tm    map[SessionID]Session\n}",
    "Related_methods": [
        {
            "Name": "Store",
            "Description": "Store is used to lock the shard before storing the session with a specific ID, and then unlock it after the operation is complete."
        },
        {
            "Name": "Delete",
            "Description": "Delete is used to remove the session identified by its ID from the shard, ensuring thread safety by locking before deletion and unlocking after."
        },
        {
            "Name": "Load",
            "Description": "Load function is used to retrieve the session from the shard using the given session ID, and returns the session along with a boolean value indicating whether the session is found."
        }
    ],
    "Related_types": [
        {
            "Name": "Session",
            "Description": "The defined type is an interface named 'Session' that outlines the structure of session management. It includes three methods: 'IsValid' checks if the session is currently valid and returns a boolean value; 'Get' retrieves the value of a specified key, both key and value are of type interface, allowing values and keys of different types to be used; 'WithValue' sets the value of a specific key and returns a new session instance reflecting this change, effectively allowing the session to be modified while maintaining immutability. This interface has no related methods or types."
        },
        {
            "Name": "SessionID",
            "Description": "The type named SessionID is defined as an unsigned 64-bit integer. This type has no specified related methods or related types."
        }
    ]
}

## Output
The shard type is a structure that contains a read/write lock and a mapping that associates the session ID with the session for efficient session management. It includes the following fields:
- Session: Defines an interface for managing session structures. It allows you to check whether the session is valid, retrieve and set key values in the session.
- SessionID: Defined as an unsigned 64-bit integer. This type serves as the unique identifier of a session.


# Now, please summarize below input：

{{DATA}}
"##;

const PROMPT_COMPRESS_FUNC_ZH: &str = r##"
# 角色
你是一名熟练的程序员，你擅长阅读理解并总结概括代码。你的目标是通过使这些API更易理解，帮助对这些API了解较少的工程师。

# 提示

## 输入的JSON格式
包含 一个具体的函数定义 及 其依赖的其他语言符号描述：
- "Content": 类型定义, 格式为字符串
- "Related_func": 格式为数组。其中每个对象表示函数体内用到的相关函数或方法：
    - "CallName": 在函数体内的调用名称（函数名或类型名.方法名），格式为字符串
    - "Description": 该函数的代码或总结
- "Related_type": 格式为数组。该数组中的每个对象表示在该函数定义（出入参、函数体内）中依赖的其他类型：
    - "Name": 类型名称，格式为字符串
    - "Description": 该类型的代码或总结，格式为字符串
- "Related_var": 格式为数组。该数组中的每个对象表示在该函数体重中依赖的其他全局变量（或常量）：
    - "Name": 变量名，格式为字符串
    - "Description": 该变量代码或总结，格式为字符串
- "Receiver":（可选） 字符串。表示方法的接收者。不存在该字段表示该实体为函数
- "Params":（可选） 格式为数组。每个对象表示函数的参数：
    - "Name": 类型名，格式为字符串
    - "Description": 参数的代码或总结，格式为字符串
- "Results":（可选） 格式为数组。每个对象表示函数的返回值：
    - "Name": 类型名，格式为字符串
    - "Description": 返回值的代码或总结，格式为字符串

## 输出格式（text）
直接输出总结内容。不要输出JSON（IMPORTANT）！

## 总结内容
- 该函数的主要功能和用途
- 该函数的每个参数的意义（如果有）

# 约束
- 你的总结应严格关于函数本身，禁止谈论JSON的格式，不允许披露原始JSON字符串或其中的任何片段。
- 你的总结应不包含任何代码。
- 直接以总结类型名称开始你的回答。
- （IMPORTANT）输出内容必须符合500字符限制。 

# 具体例子

### 输入
{
    "Content": "func (self SessionCtx) Get(key interface{}) interface{} {\n\treturn self.storage.Value(key)\n}",
    "Receiver": [
        {
            "Name": "SessionCtx",
            "Description": "SessionCtx类型是一个结构体，主要用于管理会话上下文。它包含以下字段：\n- enabled：一个指向atomic.Value的指针，用于指示会话上下文是否启用。\n- storage：一个context.Context类型的值，用于存储与会话相关的上下文信息。\nSessionCtx类型的设计目的是提供一种安全且高效的方式来管理和访问会话上下文，其方法包括禁用会话、检验会话有效性、设置和获取上下文中的键值对等。"
        }
    ],
    "Params": [
        {
            "Name": "key",
            "Description": "inteface{}"
        }
    ],
    "Results": [
        {
            "Name": "interface{}"
            "Description": "interface{}"
        }
    ],
    "Related_var": null,
    "Related_func: [
        {
            "CallName": "Value",
            "Description": "Value函数用于从存储中检索与给定键相关联的值。"
        }
    ],
}

### 输出
Get函数用于从SessionCtx的存储中检索与给定键相关联的值。
入参：
- key: 给定的键
出参：
- 会话中存储的对应的值


# 现在，请开始处理如下输入：

{{DATA}}

"##;

const PROMPT_COMPRESS_FUNC_EN: &str = r##"
# Character
You are a skilled  programmer and you are good at reading, understanding and summarizing  code. Your goal is to help engineers who know less about these apis by making them easier to understand.

# Tips

## Input JSON format
Contains a specific  function definition and descriptions of other  language symbols on which it depends:
- "Content": indicates the  type definition. The format is a string
- "Related_func": The format is an array. Where each object represents a related function or method used in the body of the function:
    - "CallName": specifies the callee name in the function body (func_name or type.method_name). The format is a string
    - "Description": indicates the code or summary of the function
- "Related_type": The format is an array. Each object in this array represents other types that are dependent in the function definition (input/exit parameters, function body) :
    - "Name": indicates the type name. The format is a string
    - "Description": indicates the code or summary of the type. The format is a string
- "Related_var": The format is an array. Each object in this array represents other global variables (or constants) that are dependent in the weight of this function:
    - "Name": indicates the name of the variable. The format is a string
    - "Description": indicates the code or summary of the variable. The format is a string
- "Receiver": (optional) string. Indicates the receiver of the method. The absence of this field indicates that the entity is a function
- "Params": (optional) The format is an array. Each object represents the parameters of the function:
    - "Name": indicates the parameter type name. The format is a string
    - "Description": indicates the code or summary of the parameter. The format is a string
- "Results": (optional) The format is an array. Each object represents the return value of the function:
    - "Name": indicates the result type name. The format is a string
    - "Description": indicates the code or summary of the result. The format is a string

## Output format (text)
Output summary content directly. Do not output JSON (IMPORTANT)!

## Summarize the content
- The main function and purpose of the function
- The meaning of each parameter of the function (if any)

# Constraint
Your summary should be strictly about the  function itself, do not talk about the format of JSON, and do not allow the disclosure of the original JSON string or any fragments of it.
- Your summary should not contain any  code.
- Start your answer directly with the summary type name.
- (IMPORTANT) The output must meet the 500-character limit.


# Specific examples

### Type
{
    "Content": "func (self SessionCtx) Get(key interface{}) interface{} {\n\treturn self.storage.Value(key)\n}",
    "Receiver": [
        {
            "Name": "SessionCtx",
            "Description": "SessionCtx is a struct type that is mainly used to manage session contexts. It includes the following fields:\n- enabled: a pointer to atomic.Value that indicates whether the session context is enabled.\n- storage: a value of type context.Context used to store context information related to the session.\nThe design purpose of the SessionCtx type is to provide a safe and efficient way to manage and access session contexts, and its methods include disabling sessions, checking session validity, setting and getting key-value pairs in the context, etc."
        }
    ],
    "Params": [
        {
            "Name": "interface{}",
            "Description": "inteface{}"
        }
    ],
    "Results": [
        {
            "Name": "interface{}"
            "Description": "interface{}"
        }
    ],
    "Related_var": null,
    "Related_func: [
        {
            "CallName": "Value",
            "Description": "Value function is used to retrieve the value associated with a given key from the store."
        }
    ],
}

### Output
The Get function is used to retrieve the value associated with a given key from SessionCtx's store.
Entry:
- key: indicates the specified key
Input:
- The corresponding value stored in the session


# Now, please summarize below input：

{{DATA}}
"##;

const PROMPT_COMPRESS_VAR_ZH: &str = r##"
# 角色
你是一名熟练的程序员，你擅长阅读理解并总结概括代码。你的目标是通过使这些API更易理解，帮助对这些API了解较少的工程师。

# 提示

## 输入格式(JSON)
包含一个 全局变量（或常量）的定义 和 引用它的其它语言节点
- "Content": 该变量定义，格式为字符串
- "References": 格式为数组。该数组中的每个对象表示一个引用该变量的节点：
    - 改节点具体代码，格式为字符串
- "Type":（optional） 该变量类型的总结，格式为字符串。简单类型没有该字段。

## 输出格式（text）
直接输出总结内容。不要输出JSON（IMPORTANT）！

## 总结内容
- 该变量的主要功能和用途
- 该变量关联的主要函数或类型（如果有）


# 约束
- 专注于总结lang包的基本功能，避免深入具体实现细节。
- 编写简短且易于理解的总结，供其他工程师参考。
- 保持与提供的输入数据一致的技术术语。
- 输出字符限制为500字符。

# 示例

## 输入
{
    "Content": "var bufferSizeLimit Integer = 1024",
    "Type": "Integer是int整型的别名"
    "Reference": [
        "func MakeBuffer(size int) []byte { if size > bufferSizeLimit { panic("over limit!") } return make([]byte, size) }",
        "func SetBufferSizeLimit(limit int) []byte { bufferSizeLimit = limit }"
    ]
}

## 输出
bufferSizeLimit是一个整数变量，初始值为整型1024，用于限制缓冲区的大小。


# 现在，请开始处理如下输入：

{{DATA}}

"##;

const PROMPT_COMPRESS_VAR_EN: &str = r##"
# Character
You are a skilled  programmer and you are good at reading, understanding and summarizing  code. Your goal is to help engineers who know less about these apis by making them easier to understand.

# Tips

## Input format (JSON)
Contains the definition of a global variable (or constant) and other language nodes that reference it
- "Content": This variable is defined in the format of a string
- "References": The format is an array. Each object in the array represents a node that references the variable:
- Change the node code to a string format
- "Type": (optional) Summary of the variable type. The format is a string. Simple types do not have this field.

## Output format (text)
Output summary content directly. Do not output JSON (IMPORTANT)!

## Summarize the content
- The main function and use of the variable
- The main function or type associated with the variable (if any)


# Constraint
- Focus on summarizing the basic features of the lang package and avoid delving into specific implementation details.
- Write short and easy to understand summaries for other engineers to refer to.
- Technical terms that are consistent with the input data provided.
- The output character limit is 500 characters.


# Examples

## Input
{
    "Content": "var bufferSizeLimit Integer = 1024",
    "Type": "Integer is an alias for the int type"
    "Reference": [
        "func MakeBuffer(size int) []byte { if size > bufferSizeLimit { panic("over limit!") } return make([]byte, size) }",
        "func SetBufferSizeLimit(limit int) []byte { bufferSizeLimit = limit }"
    ]
}

## Output
bufferSizeLimit is an integer variable with an initial value of integer 1024 that is used to limit the size of the buffer.


# Now, please summarize below input：

{{DATA}}

"##;

const PROMPT_COMPRESS_PKG_ZH: &str = r##"
# 角色
你是一名经验丰富的工程师，专门研究lang，并深入了解其各种包。你的主要职责是利用其他开发人员提供的有关公共函数和类型的数据，简化并总结lang包的基本功能。你的目标是通过使这些包更易理解，帮助对这些包了解较少的工程师。

# 提示

## 输入格式(JSON)
包含 一个具体的类型定义 及 其依赖的其他语言符号描述：
- "PkgPath": 改包的import路径, 格式为字符串
- "Functions": 格式为数组。其中每个对象表示此包内定义的公开函数和方法：
    - "Name": 函数名，格式为字符串
    - "Description": 该方法的代码或总结, 格式为字符串
- "Types": 格式为数组。该数组中的每个对象表示此包内定义的公开类型：
    - "Name": 使用该类型的名称，格式为字符串
    - "Description": 该类型的代码或总结，格式为字符串
- "Variables": 格式为数组。该数组中的每个对象表示在该包中定义的全局变量（或常量）：
    - "Name": 变量名，格式为字符串
    - "Description": 该变量代码或总结，格式为字符串

## 输出格式（text）
直接输出总结内容。不要输出JSON（IMPORTANT）！

## 总结内容
- 该包的主要功能和用途
- 该包的一些关键函数和类型的描述


# 约束
- 专注于总结lang包的基本功能，避免深入具体实现细节。
- 编写简短且易于理解的总结，供其他工程师参考。
- 保持与提供的输入数据一致的技术术语。
- 输出字符限制为2000字符。

# 示例

## 输入
{
    "PkgPath": "github.com/cloudwego/localsession/backup",
    "Functions": [
        {
            "Description": "BackupCtx用于创建一个新的会话上下文，将会话标记为启用，并将其与给定的上下文关联。然后检查是否存在默认会话管理器，如果存在，则将创建的会话绑定到会话管理器。此过程包括根据会话ID和分片编号确定会话的适当分片，将会话存储在该分片中，并在会话管理器的选项启用时异步传输会话ID。它还涉及创建或更新标签映射以确保会话ID标签的唯一性，并使用新或修改的映射更新Pprof标签。此外，通过在存储会话之前锁定分片并在存储后解锁来保证线程安全。",
            "Name": "BackupCtx",
        },
        {
            "Description": "RecoverCtxOnDemands用于根据需求使用备份处理程序恢复或更新上下文。它首先检查处理程序是否为nil，如果是，则返回未更改的当前上下文。该函数尝试使用CurSession检索当前会话，CurSession确定是否存在默认会话管理器并获取当前会话。如果未找到当前会话或其类型为SessionCtx，则函数返回未更改的上下文。然后使用Export方法从SessionCtx实例中检索存储并将其作为上下文返回。此存储与当前上下文一起传递给用户定义的处理程序，以可能生成新上下文并决定是否需要备份操作。如果不需要备份，则返回原始上下文。如果在预上下文中存在持久化的元信息值，函数将这些值合并到新或现有上下文中，优先考虑传入上下文而不是会话数据。这种双向合并确保所有持久化的元信息从先前的上下文传递到新或现有的上下文中。",
            "Name": "RecoverCtxOnDemands",
        },
        {
            "Description": "DefaultOptions用于初始化具有默认值的Options结构，包括将Enable设置为false，并使用DefaultManagerOptions方法初始化ManagerOptions，分片数为100，垃圾收集间隔为10分钟，并禁用隐式异步传输。",
            "Name": "DefaultOptions",
        },
        {
            "Description": "ClearCtx用于从默认会话管理器的分片中删除指定的会话ID（如果存在），通过检查当前例程的ID，根据该ID确定相关分片，并在找到时安全地删除会话。这还包括在启用跟踪时从Pprof标签中清除会话ID。",
            "Name": "ClearCtx",
        },
        {
            "Description": "Init用于在启用选项时初始化会话管理器。它通过基于环境变量配置管理器的选项，终止任何现有会话以防止重叠，并创建一个新的SessionManager实例来实现。这包括初始化分片，确保分片数大于零，并在必要时启动垃圾收集以确保高效的会话管理并防止并发执行。",
            "Name": "Init",
        }
    ],
    "Types": [
        {
            "Description": "\"Options\"结构包括一个名为\"Enable\"的布尔字段，并嵌入了本地会话包中的\"ManagerOptions\"。\"ManagerOptions\"专门用于指示会话管理的细节和goroutines的行为。它包括三个重要字段：\"EnableImplicitlyTransmitAsync\"、\"ShardNumber\"和\"GCInterval\"。\"EnableImplicitlyTransmitAsync\"字段是一个布尔值，便于将当前会话无缝传输到子goroutines，尽管它需要与`pprof.Do()`进行精确互动以正确操作。\"ShardNumber\"是一个整数，影响会话ID的分布，要求值大于零。\"GCInterval\"确定SessionManager中的垃圾收集频率，持续时间大于一秒钟会激活垃圾收集，值为零则关闭垃圾收集。此结构没有任何相关方法或类型。",
            "Name": "Options",
        },
        {
            "Description": "类型`BackupHandler`是一个函数类型，接受两个参数，都是`context.Context`类型，分别表示先前和当前的上下文。它返回两个值：一个新的`context.Context`类型的上下文和一个布尔值，指示是否需要备份。此类型没有相关的方法或类型。",
            "Name": "BackupHandler",
        }
    ],
    "Variables": [
        {
            "Description": "DefaultManager是一个全局变量，用于存储默认的会话管理器。它是一个指向`SessionManager`类型的指针，用于管理会话的创建、备份、恢复和清除。此变量没有相关的方法或类型。",
            "Name": "DefaultManager",
        }
    ]
}

## 输出
此包位于github.com/cloudwego/localsession/backup，为应用中的会话管理提供工具，特别关注会话上下文的备份和恢复机制。它包括创建、备份、恢复和清除会话上下文的功能，以及使用默认或自定义设置初始化会话管理。
关键函数：
    - BackupCtx(ctx context.Context): 创建一个新的会话上下文，使其启用并将其与给定的上下文关联。它确保线程安全，并在配置时异步传输会话ID。
    - RecoverCtxOnDemands(ctx context.Context, handler BackupHandler) context.Context: 使用备份处理程序根据需求恢复或更新上下文，能够合并先前上下文中的持久化元信息。
    - Init(opts Options): 使用指定的选项初始化会话管理器，确保高效的会话管理和垃圾收集。
关键类型：
    - Options: 一个结构体，包括会话管理的设置，例如启用隐式异步传输、分片数量和垃圾收集间隔。
    - BackupHandler: 一种函数类型，接受两个上下文（先前和当前），返回一个新的上下文和一个布尔值，指示是否需要备份。
此包旨在通过提供强大的会话备份和恢复机制、自定义的会话管理选项，并确保线程安全和高效的资源管理，增强应用中的会话管理。
关键全局变量：
    - DefaultManager: 一个指向SessionManager类型的指针，用于管理会话的创建、备份、恢复和清除。


# 现在，请开始处理如下输入：

{{DATA}}

"##;

const PROMPT_COMPRESS_PKG_EN: &str = r##"
# Character
You are an experienced engineer who specializes in lang and has in-depth knowledge of its various packages. Your primary responsibility is to simplify and summarize the basic functionality of the lang package using data provided by other developers about common functions and types. Your goal is to help engineers who know less about these packages by making them easier to understand.

# Tips

## Input format (JSON)
Contains a specific  type definition and descriptions of other  language symbols that depend on it:
- "PkgPath": indicates the import path of the package. The format is a string
- "Functions": The format is an array. Where each object represents the public functions and methods defined within this package:
- "Name": indicates the function name. The format is a string
- "Description": indicates the code or summary of the method. The format is a string
- "Types": The format is an array. Each object in this array represents a public type defined within this package:
- "Name": specifies the name of the type. The format is a string
- "Description": indicates the code or summary of the type. The format is a string
- "Variables": Format is an array. Each object in this array represents a global variable (or constant) defined in that package:
- "Name": indicates the name of the variable. The format is a string
- "Description": indicates the code or summary of the variable. The format is a string

## Output format (text)
Output summary content directly. Do not output JSON (IMPORTANT)!

## Summarize the content
- The main functions and uses of the package
- Description of some of the key functions and types of the package


# Constraint
- Focus on summarizing the basic features of the lang package and avoid delving into specific implementation details.
- Write short and easy to understand summaries for other engineers to refer to.
- Technical terms that are consistent with the input data provided.
- The output character limit is 2000 characters.

# Examples

## Input
{
    "PkgPath": "github.com/cloudwego/localsession/backup",
    "Functions": [
        {
            "Description": "BackupCtx is used to create a new session context, mark the session as enabled, and associate it with the given context. It ensures thread safety",
            "Name": "BackupCtx",
        },
        {
            "Description": "RecoverCtxOnDemands is used to recover or update the context on demand using a backup handler, allowing you to merge persistent meta-information from previous contexts.",
            "Name": "RecoverCtxOnDemands",
        },
        {
            "Description": "DefaultOptions is used to initialize the Options structure with default values, including setting Enable to false and initializing ManagerOptions with DefaultManagerOptions method.",
            "Name": "DefaultOptions",
        },
        {
            "Description": "ClearCtx is used to remove the specified session ID from the shards of the default session manager, ensuring thread safety.",
            "Name": "ClearCtx",
        },
        {
            "Description": "Init is used to initialize the session manager when options are enabled.",
            "Name": "Init",
        }
    ],
    "Types": [
        {
            "Description": "Options structure includes a boolean field named \"Enable\" and embeds \"ManagerOptions\" from the local session package.",
            "Name": "Options",
        },
        {
            "Description": "BackupHandler type is a function type that accepts two contexts (previous and current) and returns a new context and a boolean value indicating whether a backup is required.",
            "Name": "BackupHandler",
        }
    ],
    "Variables": [
        {
            "Description": "DefaultManager is a global variable used to store the default session manager.",
            "Name": "DefaultManager",
        }
    ]
}

## Output
This package is located in github.com/cloudwego/localsession/backup, to provide tools to  application session management, pay special attention to the backup and restore mechanism of session context. It includes the ability to create, back up, restore, and clear session context, as well as initialize session management with default or custom Settings.
Key functions:
- BackupCtx(ctx context.context): Creates a new session context, enables it and associates it with the given Context. It ensures thread-safe and asynchronously transfers session ids when configured.
- RecoverCtxOnDemands(ctx context.Context, handler BackupHandler) context.Context:  Use a backup handler to restore or update the context on demand, allowing you to merge persistent meta-information from previous contexts.
- Init(opts Options): Initializes the session manager with the specified options, ensuring efficient session management and garbage collection.
Key types:
- Options: A structure that includes Settings for session management, such as enabling implicit asynchronous transfer, number of shards, and garbage collection interval.
- BackupHandler: A function type that accepts two contexts (previous and current) and returns a new context and a Boolean value indicating whether a backup is required.
This package is designed to enhance session management in  applications by providing a robust session backup and recovery mechanism, custom session management options, and ensuring thread-safe and efficient resource management.
Key global variables:
- DefaultManager: A pointer to the SessionManager type, which is used to manage session creation, backup, recovery, and clearing.


# Now, please summarize below input：

{{DATA}}

"##;
