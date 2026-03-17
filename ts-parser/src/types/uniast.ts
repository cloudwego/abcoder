// UNIAST v0.1.3 TypeScript interfaces
// Based on the specification in spec.md

// ================================================================
// Core Structures
// ================================================================

/**
 * Root object representing an entire code repository.
 */
export interface Repository {
  /** Unique identifier for the repository. Field name in JSON is "id". */
  id: string;
  /** UNIAST specification version. Fixed to "v0.1.3". */
  ASTVersion: string;
  /** abcoder version used to parse the repository. Field name in JSON is "ToolVersion". */
  ToolVersion: string;
  /** File directory of the repository, usually should be an absolute path. */
  Path: string;
  /** Map of all modules in the repository. Keys are unique path identifiers. */
  Modules: Record<string, Module>;
  /**
   * Global symbol graph. Keys are fully-qualified unique symbol strings, values are the corresponding Node objects.
   */
  Graph: Record<string, Node>;
}

/**
 * A compilation unit, e.g., a Go module or an npm package.
 */
export interface Module {
  Language: 'go' | 'rust' | 'cxx' | 'python' | 'typescript' | '';
  Version: string;
  Name: string;
  /** Path relative to the repository root. Empty string "" denotes an external dependency module. */
  Dir: string;
  /** Map of all packages in the module. Keys are package import paths. */
  Packages: Record<string, Package>;
  /** (Optional) Map of module dependencies. */
  Dependencies?: Record<string, string>;
  /** (Optional) Map of metadata for all files in the module. */
  Files?: Record<string, File>;
}

/**
 * A namespace containing a group of code symbols.
 */
export interface Package {
  IsMain: boolean;
  IsTest: boolean;
  /** Unique import path for this package. Located at the top level of the Package object. */
  PkgPath: string;
  /** Map of all functions and methods. Keys are symbol names. */
  Functions: Record<string, Function>;
  /** Map of all type definitions. Keys are type names. */
  Types: Record<string, Type>;
  /** Map of all global variables and constants. Keys are variable names. */
  Vars: Record<string, Var>;
}

// ================================================================
// Core Definitions
// ================================================================

/**
 * Globally unique identifier for any code symbol.
 */
export interface Identity {
  ModPath: string;
  PkgPath: string;
  Name: string;
}

/**
 * Exact location of a symbol definition or reference in a source file.
 */
export interface FileLine {
  File: string;
  Line: number;       // 1-based line number.
  StartOffset: number;
  EndOffset: number;
}

/**
 * Reference to another code symbol. Contains the target symbol's Identity and the location of the reference.
 * Note: the FileLine part is optional.
 */
export type Dependency = Identity & Partial<FileLine>;

// ================================================================
// Concrete Symbol Structures
// ================================================================

export interface Function {
  // --- Identity & FileLine (inline fields) ---
  ModPath: string;
  PkgPath: string;
  Name: string;
  File: string;
  Line: number;
  StartOffset: number;
  EndOffset: number;

  // --- Function-specific Fields ---
  Exported: boolean;
  IsMethod: boolean;
  IsInterfaceMethod: boolean;
  /** Complete source code of the function, including signature and body. */
  Content: string;
  Signature?: string;
  Receiver?: Receiver;
  Params?: Dependency[];
  Results?: Dependency[];
  FunctionCalls?: Dependency[];
  MethodCalls?: Dependency[];
  Types?: Dependency[];
  /** References to package-level variables, exported or not. */
  GlobalVars?: Dependency[];
}

export interface Type {
  // --- Identity & FileLine (inline fields) ---
  ModPath: string;
  PkgPath: string;
  Name: string;
  File: string;
  Line: number;
  StartOffset: number;
  EndOffset: number;

  // --- Type-specific Fields ---
  Exported: boolean;
  TypeKind: 'struct' | 'interface' | 'typedef' | 'enum';
  Content: string;
  SubStruct?: Dependency[];
  InlineStruct?: Dependency[];
  Methods?: Record<string, Identity>;
  Implements?: Identity[];
}

export interface Var {
  // --- Identity & FileLine (inline fields) ---
  ModPath: string;
  PkgPath: string;
  Name: string;
  File: string;
  Line: number;
  StartOffset: number;
  EndOffset: number;

  // --- Var-specific Fields ---
  IsExported: boolean;
  IsConst: boolean;
  IsPointer: boolean;
  Content: string;
  Type?: Identity;
  Dependencies?: Dependency[];
  Groups?: Identity[];
}

// ================================================================
// Auxiliary & Graph Structures
// ================================================================

/**
 * Represents a node (symbol entity) in the code.
 */
export interface Node {
  // --- Identity (inline fields) ---
  ModPath: string;
  PkgPath: string;
  Name: string;

  // --- Node-specific Fields ---
  Type: 'FUNC' | 'TYPE' | 'VAR' | 'UNKNOWN';
  /** (Optional) List of other nodes this node depends on (outgoing edges). */
  Dependencies?: Relation[];
  /** (Optional) List of nodes that reference this node (incoming edges). */
  References?: Relation[];
  /** (Optional) List of interface nodes this node implements. */
  Implements?: Relation[];
  /** (Optional) List of parent nodes this node inherits from. */
  Inherits?: Relation[];
  /** (Optional) List of other nodes in the same definition group. */
  Groups?: Relation[];
}

/**
 * Describes a relationship between two nodes.
 */
export interface Relation {
  // --- Identity (inline fields) ---
  ModPath: string;
  PkgPath: string;
  Name: string;

  // --- Relation-specific Fields ---
  Kind: 'Dependency' | 'Implement' | 'Inherit' | 'Group';
}

export interface File {
  Path: string;
  Imports?: Import[];
  Package?: string;
}

/**
 * Represents an import declaration. Can be a simple string or an object containing alias and path.
 */
export type Import = string | {
  Alias?: string;
  Path: string;
};

export interface Receiver {
  IsPointer: boolean;
  Type: Identity;
}