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

// Package pb contains message definitions for Java Parser IPC using JSON serialization.
package pb

// ==================== 顶层消息包装器 ====================

const (
	TYPE_ANALYZE_REQUEST  = "analyze_request"
	TYPE_ANALYZE_RESPONSE = "analyze_response"
	TYPE_STOP_REQUEST     = "stop_request"
	TYPE_HEARTBEAT        = "heartbeat"
)

// Message 是所有 IPC JSON 消息的统一包装。
// 说明：Java 侧会在 payload 里再嵌一层具体消息体（例如 AnalyzeResponse）。
type Message struct {
	Type      string      `json:"type"`
	RequestId string      `json:"requestId,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
}

func (m *Message) GetAnalyzeRequest() *AnalyzeRequest {
	if m == nil {
		return nil
	}
	if v, ok := m.Payload.(*AnalyzeRequest); ok {
		return v
	}
	return nil
}

func (m *Message) GetAnalyzeResponse() *AnalyzeResponse {
	if m == nil {
		return nil
	}
	if v, ok := m.Payload.(*AnalyzeResponse); ok {
		return v
	}
	return nil
}

func (m *Message) GetStopRequest() *StopRequest {
	if m == nil {
		return nil
	}
	if v, ok := m.Payload.(*StopRequest); ok {
		return v
	}
	return nil
}

func (m *Message) GetHeartbeat() *Heartbeat {
	if m == nil {
		return nil
	}
	if v, ok := m.Payload.(*Heartbeat); ok {
		return v
	}
	return nil
}

// ==================== 类型枚举（JSON string） ====================

// ClassType enum
type ClassType string

const (
	ClassType_CLASS_TYPE_UNKNOWN    ClassType = ""
	ClassType_CLASS_TYPE_CLASS      ClassType = "class"
	ClassType_CLASS_TYPE_INTERFACE  ClassType = "interface"
	ClassType_CLASS_TYPE_ENUM       ClassType = "enum"
	ClassType_CLASS_TYPE_ANNOTATION ClassType = "annotation"
	ClassType_CLASS_TYPE_RECORD     ClassType = "record"
)

func (x ClassType) String() string {
	switch x {
	case ClassType_CLASS_TYPE_CLASS:
		return "CLASS"
	case ClassType_CLASS_TYPE_INTERFACE:
		return "INTERFACE"
	case ClassType_CLASS_TYPE_ENUM:
		return "ENUM"
	case ClassType_CLASS_TYPE_ANNOTATION:
		return "ANNOTATION"
	case ClassType_CLASS_TYPE_RECORD:
		return "RECORD"
	default:
		return "UNKNOWN"
	}
}

// SourceType enum
type SourceType string

const (
	SourceType_SOURCE_TYPE_UNKNOWN      SourceType = "unknown"
	SourceType_SOURCE_TYPE_LOCAL        SourceType = "local"
	SourceType_SOURCE_TYPE_MAVEN        SourceType = "maven"
	SourceType_SOURCE_TYPE_EXTERNAL_JAR SourceType = "external_jar"
	SourceType_SOURCE_TYPE_JDK          SourceType = "jdk"
)

// DependencyDepth enum
type DependencyDepth string

const (
	DependencyDepth_DEPTH_UNKNOWN    DependencyDepth = "unknown"
	DependencyDepth_DEPTH_LOCAL      DependencyDepth = "local"
	DependencyDepth_DEPTH_DIRECT     DependencyDepth = "direct"
	DependencyDepth_DEPTH_TRANSITIVE DependencyDepth = "transitive"
	DependencyDepth_DEPTH_JDK        DependencyDepth = "jdk"
)

// DependencyKind 在新协议中为 string：import/extends/implements/field/method_param/method_return/method_call/annotation
type DependencyKind string

const (
	DependencyKind_DEP_KIND_UNKNOWN       DependencyKind = ""
	DependencyKind_DEP_KIND_IMPORT        DependencyKind = "import"
	DependencyKind_DEP_KIND_EXTENDS       DependencyKind = "extends"
	DependencyKind_DEP_KIND_IMPLEMENTS    DependencyKind = "implements"
	DependencyKind_DEP_KIND_FIELD         DependencyKind = "field"
	DependencyKind_DEP_KIND_METHOD_PARAM  DependencyKind = "method_param"
	DependencyKind_DEP_KIND_METHOD_RETURN DependencyKind = "method_return"
	DependencyKind_DEP_KIND_METHOD_CALL   DependencyKind = "method_call"
	DependencyKind_DEP_KIND_ANNOTATION    DependencyKind = "annotation"
)

// ErrorCode 在新协议中为 string：repo_not_found/parse_failed/config_invalid/io/timeout
type ErrorCode string

const (
	ErrorCode_ERROR_UNKNOWN        ErrorCode = ""
	ErrorCode_ERROR_REPO_NOT_FOUND ErrorCode = "repo_not_found"
	ErrorCode_ERROR_PARSE_FAILED   ErrorCode = "parse_failed"
	ErrorCode_ERROR_CONFIG_INVALID ErrorCode = "config_invalid"
	ErrorCode_ERROR_IO             ErrorCode = "io"
	ErrorCode_ERROR_TIMEOUT        ErrorCode = "timeout"
)

// AnalyzerConfig holds configuration for the analyzer
type AnalyzerConfig struct {
	ResolveMavenDependencies bool     `json:"resolveMavenDependencies,omitempty"`
	M2RepositoryPath         string   `json:"m2RepositoryPath,omitempty"`
	ExtraJarPaths            []string `json:"extraJarPaths,omitempty"`
	IncludeExternalClasses   bool     `json:"includeExternalClasses,omitempty"`
}

// AnalyzeRequest is the request message sent to Java parser
type AnalyzeRequest struct {
	RequestId string          `json:"requestId"`
	RepoPath  string          `json:"repoPath"`
	Config    *AnalyzerConfig `json:"config,omitempty"`
}

// StopRequest is the stop request message
type StopRequest struct {
	RequestId string `json:"requestId"`
}

// Heartbeat is heartbeat message
type Heartbeat struct {
	Timestamp int64 `json:"timestamp"`
}

// ProgressUpdate contains progress information
type ProgressUpdate struct {
	Percentage     int32  `json:"percentage"`
	Phase          string `json:"phase"`
	Message        string `json:"message,omitempty"`
	ProcessedFiles int32  `json:"processedFiles,omitempty"`
	TotalFiles     int32  `json:"totalFiles,omitempty"`
}

// FileInfo contains information about a source file
type FileInfo struct {
	FilePath     string   `json:"filePath"`
	AbsolutePath string   `json:"absolutePath,omitempty"`
	ClassNames   []string `json:"classNames,omitempty"`
	FileSize     int64    `json:"fileSize,omitempty"`
	LineCount    int32    `json:"lineCount,omitempty"`
}

// SourceInfo contains source information for a class
type SourceInfo struct {
	Type            SourceType      `json:"type"`
	MavenCoordinate string          `json:"mavenCoordinate,omitempty"`
	JarPath         string          `json:"jarPath,omitempty"`
	Depth           DependencyDepth `json:"depth,omitempty"`
}

// DependencyInfo contains dependency information
type DependencyInfo struct {
	ClassName       string          `json:"className"`
	SourceType      SourceType      `json:"sourceType,omitempty"`
	Depth           DependencyDepth `json:"depth,omitempty"`
	MavenCoordinate string          `json:"mavenCoordinate,omitempty"`
	Kind            DependencyKind  `json:"kind,omitempty"`
}

// ==================== ClassInfo 细节结构（新协议） ====================

// SymbolText 表示符号原文 + 位置（用于 extends/implements 等）。
type SymbolText struct {
	Fqcn        string `json:"fqcn,omitempty"`
	RawText     string `json:"rawText,omitempty"`
	StartLine   int32  `json:"startLine,omitempty"`
	StartColumn int32  `json:"startColumn,omitempty"`
	EndLine     int32  `json:"endLine,omitempty"`
	EndColumn   int32  `json:"endColumn,omitempty"`
}

// FieldDetail 字段明细。
type FieldDetail struct {
	Name        string `json:"name,omitempty"`
	TypeFqcn    string `json:"typeFqcn,omitempty"`
	TypeRawText string `json:"typeRawText,omitempty"`
	RawText     string `json:"rawText,omitempty"`
	StartLine   int32  `json:"startLine,omitempty"`
	StartColumn int32  `json:"startColumn,omitempty"`
	EndLine     int32  `json:"endLine,omitempty"`
	EndColumn   int32  `json:"endColumn,omitempty"`
}

// ReturnTypeDetail 返回值类型明细。
type ReturnTypeDetail struct {
	TypeFqcn    string `json:"typeFqcn,omitempty"`
	TypeRawText string `json:"typeRawText,omitempty"`
	StartLine   int32  `json:"startLine,omitempty"`
	StartColumn int32  `json:"startColumn,omitempty"`
	EndLine     int32  `json:"endLine,omitempty"`
	EndColumn   int32  `json:"endColumn,omitempty"`
}

// ParameterDetail 参数明细。
type ParameterDetail struct {
	Name        string `json:"name,omitempty"`
	TypeFqcn    string `json:"typeFqcn,omitempty"`
	TypeRawText string `json:"typeRawText,omitempty"`
	RawText     string `json:"rawText,omitempty"`
	StartLine   int32  `json:"startLine,omitempty"`
	StartColumn int32  `json:"startColumn,omitempty"`
	EndLine     int32  `json:"endLine,omitempty"`
	EndColumn   int32  `json:"endColumn,omitempty"`
}

// MethodDetail 方法/构造器明细。
type MethodDetail struct {
	Descriptor  string             `json:"descriptor,omitempty"`
	RawText     string             `json:"rawText,omitempty"`
	StartLine   int32              `json:"startLine,omitempty"`
	StartColumn int32              `json:"startColumn,omitempty"`
	EndLine     int32              `json:"endLine,omitempty"`
	EndColumn   int32              `json:"endColumn,omitempty"`
	ReturnType  *ReturnTypeDetail  `json:"returnType,omitempty"`
	Parameters  []*ParameterDetail `json:"parameters,omitempty"`
	MethodCalls []*MethodCallInfo  `json:"methodCalls,omitempty"`
}

// GetName 兼容旧 converter：优先使用 Name，否则从 Descriptor/Signature 做降级提取。
func (m *MethodDetail) GetName() string {
	if m == nil {
		return ""
	}

	// 可能的 descriptor 形式："foo(int,java.lang.String)" 或 "<init>(...)"
	// 可能的 signature 形式："public void foo(int)"
	if m.Descriptor != "" {
		for i := 0; i < len(m.Descriptor); i++ {
			if m.Descriptor[i] == '(' {
				return m.Descriptor[:i]
			}
		}
		return m.Descriptor
	}
	return ""
}

// GetStartLine 兼容旧 converter。
func (m *MethodDetail) GetStartLine() int32 {
	if m == nil {
		return 0
	}
	if m.StartLine != 0 {
		return m.StartLine
	}
	return 0
}

// GetParameterTypes 兼容旧 converter：优先 parameters，否则返回空。
func (m *MethodDetail) GetParameterTypes() []string {
	if m == nil {
		return nil
	}
	if len(m.Parameters) == 0 {
		return nil
	}
	res := make([]string, 0, len(m.Parameters))
	for _, p := range m.Parameters {
		if p == nil {
			continue
		}
		if p.TypeFqcn != "" {
			res = append(res, p.TypeFqcn)
		}
	}
	return res
}

// GetReturnType 兼容旧 converter。
func (m *MethodDetail) GetReturnType() string {
	if m == nil {
		return ""
	}
	if m.ReturnType != nil {
		return m.ReturnType.TypeFqcn
	}
	return ""
}

// ClassInfo contains class information (core steaming unit)
type ClassInfo struct {
	ClassName   string      `json:"className"`
	PackageName string      `json:"packageName"`
	FilePath    string      `json:"filePath"`
	ClassType   ClassType   `json:"classType,omitempty"` // class/interface/enum/annotation/record
	Source      *SourceInfo `json:"source,omitempty"`

	// 类定义原文（严格按源文件 Range 截取）
	RawText string `json:"rawText,omitempty"`

	// 依赖信息
	Imports           []string          `json:"imports,omitempty"`
	ExtendsTypes      []string          `json:"extendsTypes,omitempty"`
	ImplementsTypes   []string          `json:"implementsTypes,omitempty"`
	ExtendsDetails    []*SymbolText     `json:"extendsDetails,omitempty"`
	ImplementsDetails []*SymbolText     `json:"implementsDetails,omitempty"`
	Fields            []*FieldDetail    `json:"fields,omitempty"`
	Methods           []*MethodDetail   `json:"methods,omitempty"`
	Dependencies      []*DependencyInfo `json:"dependencies,omitempty"`

	// 位置信息
	StartLine   int32 `json:"startLine,omitempty"`
	EndLine     int32 `json:"endLine,omitempty"`
	StartColumn int32 `json:"startColumn,omitempty"`
	EndColumn   int32 `json:"endColumn,omitempty"`

	// 旧协议兼容字段（不会在新协议中发送）
	Content string `json:"content,omitempty"`
}

// GetContent 兼容旧调用：优先 rawText。
func (x *ClassInfo) GetContent() string {
	if x == nil {
		return ""
	}
	if x.RawText != "" {
		return x.RawText
	}
	return x.Content
}

// MethodCallInfo contains method call information
type MethodCallInfo struct {
	CallerClass   string   `json:"callerClass,omitempty"`
	CallerMethod  string   `json:"callerMethod,omitempty"`
	CalleeClass   string   `json:"calleeClass,omitempty"`
	CalleeMethod  string   `json:"calleeMethod,omitempty"`
	ArgumentTypes []string `json:"argumentTypes,omitempty"`
	ReturnType    string   `json:"returnType,omitempty"`
	Resolved      bool     `json:"resolved,omitempty"`
	RawText       string   `json:"rawText,omitempty"`

	// 位置
	FilePath  string `json:"filePath,omitempty"`
	Line      int32  `json:"line,omitempty"`
	Column    int32  `json:"column,omitempty"`
	EndLine   int32  `json:"endLine,omitempty"`
	EndColumn int32  `json:"endColumn,omitempty"`
}

// Summary contains analysis summary
type Summary struct {
	TotalTimeMs        int64  `json:"totalTimeMs,omitempty"`
	LocalClassCount    int32  `json:"localClassCount,omitempty"`
	ExternalClassCount int32  `json:"externalClassCount,omitempty"`
	FileCount          int32  `json:"fileCount,omitempty"`
	MethodCallCount    int32  `json:"methodCallCount,omitempty"`
	DependencyCount    int32  `json:"dependencyCount,omitempty"`
	LoadTimeMs         int64  `json:"loadTimeMs,omitempty"`
	ParseTimeMs        int64  `json:"parseTimeMs,omitempty"`
	AnalyzeTimeMs      int64  `json:"analyzeTimeMs,omitempty"`
	Success            bool   `json:"success"`
	Message            string `json:"message,omitempty"`
}

// ErrorInfo contains error information
type ErrorInfo struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	FilePath   string    `json:"filePath,omitempty"`
	StackTrace string    `json:"stackTrace,omitempty"`
}

const (
	PAYLOAD_PROGRESS    = "progress"
	PAYLOAD_FILE_INFO   = "file_info"
	PAYLOAD_CLASS_INFO  = "class_info"
	PAYLOAD_METHOD_CALL = "method_call"
	PAYLOAD_SUMMARY     = "summary"
	PAYLOAD_ERROR       = "error"
)

// AnalyzeResponse 是分析响应的内层消息体。
// 外层会被 Message 包装：{"type":"analyze_response","requestId":"...","payload":{...AnalyzeResponse...}}
type AnalyzeResponse struct {
	RequestId   string      `json:"requestId"`
	PayloadType string      `json:"payloadType"` // progress/file_info/class_info/method_call/summary/error
	Payload     interface{} `json:"payload"`
}

func (x *AnalyzeResponse) GetRequestId() string {
	if x != nil {
		return x.RequestId
	}
	return ""
}

func (x *AnalyzeResponse) GetProgress() *ProgressUpdate {
	if x == nil || x.PayloadType != PAYLOAD_PROGRESS {
		return nil
	}
	if v, ok := x.Payload.(*ProgressUpdate); ok {
		return v
	}
	return nil
}

func (x *AnalyzeResponse) GetFileInfo() *FileInfo {
	if x == nil || x.PayloadType != PAYLOAD_FILE_INFO {
		return nil
	}
	if v, ok := x.Payload.(*FileInfo); ok {
		return v
	}
	return nil
}

func (x *AnalyzeResponse) GetClassInfo() *ClassInfo {
	if x == nil || x.PayloadType != PAYLOAD_CLASS_INFO {
		return nil
	}
	if v, ok := x.Payload.(*ClassInfo); ok {
		return v
	}
	return nil
}

func (x *AnalyzeResponse) GetMethodCall() *MethodCallInfo {
	if x == nil || x.PayloadType != PAYLOAD_METHOD_CALL {
		return nil
	}
	if v, ok := x.Payload.(*MethodCallInfo); ok {
		return v
	}
	return nil
}

func (x *AnalyzeResponse) GetSummary() *Summary {
	if x == nil || x.PayloadType != PAYLOAD_SUMMARY {
		return nil
	}
	if v, ok := x.Payload.(*Summary); ok {
		return v
	}
	return nil
}

func (x *AnalyzeResponse) GetError() *ErrorInfo {
	if x == nil || x.PayloadType != PAYLOAD_ERROR {
		return nil
	}
	if v, ok := x.Payload.(*ErrorInfo); ok {
		return v
	}
	return nil
}

// Getter methods for ClassInfo
func (x *ClassInfo) GetClassName() string {
	if x != nil {
		return x.ClassName
	}
	return ""
}

func (x *ClassInfo) GetFilePath() string {
	if x != nil {
		return x.FilePath
	}
	return ""
}

func (x *ClassInfo) GetClassType() ClassType {
	if x != nil {
		return x.ClassType
	}
	return ClassType_CLASS_TYPE_UNKNOWN
}

func (x *ClassInfo) GetImports() []string {
	if x != nil {
		return x.Imports
	}
	return nil
}

func (x *ClassInfo) GetExtendsTypes() []string {
	if x != nil {
		return x.ExtendsTypes
	}
	return nil
}

func (x *ClassInfo) GetImplementsTypes() []string {
	if x != nil {
		return x.ImplementsTypes
	}
	return nil
}

func (x *ClassInfo) GetMethods() []*MethodDetail {
	if x != nil {
		return x.Methods
	}
	return nil
}

func (x *ClassInfo) GetFields() []*FieldDetail {
	if x != nil {
		return x.Fields
	}
	return nil
}

func (x *ClassInfo) GetStartLine() int32 {
	if x != nil {
		return x.StartLine
	}
	return 0
}

func (x *ClassInfo) GetEndLine() int32 {
	if x != nil {
		return x.EndLine
	}
	return 0
}

// Getter methods for Summary
func (x *Summary) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *Summary) GetLocalClassCount() int32 {
	if x != nil {
		return x.LocalClassCount
	}
	return 0
}

func (x *Summary) GetFileCount() int32 {
	if x != nil {
		return x.FileCount
	}
	return 0
}

func (x *Summary) GetTotalTimeMs() int64 {
	if x != nil {
		return x.TotalTimeMs
	}
	return 0
}
