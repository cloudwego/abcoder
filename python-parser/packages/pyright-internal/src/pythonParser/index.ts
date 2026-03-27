#!/usr/bin/env node
// 入口文件 - 使用 pyright-internal 解析 Python 仓库
import * as path from 'path';
import * as fs from 'fs';

// 导出 parseRepository 函数
export { parseRepository };

// 导入 UniAST 类型定义
import { Repository, Module, Package, Function, Type, Var, Node, Dependency, Identity, Relation } from './types/uniast';

// 导入 pyright-internal 核心模块 (内部相对导入)
import { AnalyzerService } from '../analyzer/service';
import { Program } from '../analyzer/program';
import { TypeEvaluator } from '../analyzer/typeEvaluatorTypes';
import { ConfigOptions } from '../common/configOptions';
import { Uri } from '../common/uri/uri';
import { createServiceProvider } from '../common/serviceProviderExtensions';
import { RealFileSystem, createFromRealFileSystem } from '../common/realFileSystem';
import { RealTempFile } from '../common/realFileSystem';
import { StandardConsole, LogLevel } from '../common/console';
import { FullAccessHost } from '../common/fullAccessHost';
import { ParseTreeWalker } from '../analyzer/parseTreeWalker';
import {
    ParseNode,
    NameNode,
    CallNode,
    MemberAccessNode,
    ParseNodeType,
} from '../parser/parseNodes';
import { DeclarationType } from '../analyzer/declaration';
import { ClassType, TypeCategory } from '../analyzer/types';

// ================================================================
// 辅助函数：使用 pyright API 提取符号信息
// ================================================================

/**
 * 检查声明是否为类型别名
 * 基于 pyright 的 isExplicitTypeAliasDeclaration 逻辑
 * 使用 pyright API: decl.typeAnnotationNode
 */
function isTypeAliasDecl(decl: any): boolean {
    // 1. 必须是 Variable 类型
    if (decl.type !== DeclarationType.Variable) {
        return false;
    }

    // 2. 必须有类型注解节点
    if (!decl.typeAnnotationNode) {
        return false;
    }

    try {
        // 3. 检查 typeAnnotationNode 的值
        const annotationNode = decl.typeAnnotationNode;
        const annotationData = annotationNode.d || annotationNode;

        // 方法 1: 检查名称是否为 TypeAlias
        if (annotationData.value === 'TypeAlias') {
            return true;
        }

        // 方法 2: 检查成员访问 (typing.TypeAlias)
        if (annotationNode.nodeType === 34) { // MemberAccess
            const member = annotationData.member;
            if (member && member.d && member.d.value === 'TypeAlias') {
                return true;
            }
        }
    } catch (e) {
        // 忽略错误
    }

    return false;
}

/**
 * 提取类成员（方法、属性）
 * 使用 pyright API: evaluator.getTypeOfClass() + ClassType.getSymbolTable()
 */
function extractClassMembers(
    classDecl: any,
    evaluator: TypeEvaluator
): { methods: string[]; vars: string[] } {
    const members = { methods: [] as string[], vars: [] as string[] };

    if (!classDecl.node) {
        return members;
    }

    try {
        // 使用 pyright API 获取类类型
        const classType = evaluator.getTypeOfClass(classDecl.node);

        if (classType && classType.classType) {
            // 使用 pyright API 获取成员符号表
            const symbolTable = ClassType.getSymbolTable(classType.classType);

            if (symbolTable) {
                const iter = symbolTable.entries();
                while (true) {
                    const result = iter.next();
                    if (result.done) break;

                    const memberName = result.value[0];
                    const memberSymbol = result.value[1];

                    // 跳过内置属性
                    if (memberName.startsWith('__') && memberName.endsWith('__')) {
                        continue;
                    }

                    const memberDecls = memberSymbol.getDeclarations();
                    if (memberDecls.length > 0) {
                        const memberDecl = memberDecls[0];

                        if (memberDecl.type === DeclarationType.Function) {
                            members.methods.push(memberName);
                        } else if (memberDecl.type === DeclarationType.Variable) {
                            members.vars.push(memberName);
                        }
                    }
                }
            }
        }
    } catch (e) {
        // 忽略错误
    }

    return members;
}

// 扫描 Python 文件 - 支持目录和单个文件
function scanPythonFiles(targetPath: string): string[] {
    const files: string[] = [];

    try {
        const stats = fs.statSync(targetPath);

        if (stats.isFile()) {
            // 单个文件
            if (targetPath.endsWith('.py')) {
                files.push(targetPath);
            }
        } else if (stats.isDirectory()) {
            // 目录，递归扫描
            function walk(dir: string) {
                try {
                    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
                        if (entry.name.startsWith('.') || entry.name === '__pycache__') {
                            continue;
                        }
                        const fullPath = path.join(dir, entry.name);
                        if (entry.isDirectory()) {
                            walk(fullPath);
                        } else if (entry.isFile() && entry.name.endsWith('.py')) {
                            files.push(fullPath);
                        }
                    }
                } catch (e) {
                    // 忽略权限错误
                }
            }
            walk(targetPath);
        }
    } catch (e) {
        // 忽略错误
    }

    return files;
}

// 收集符号引用的 Walker - 支持函数调用、变量引用、类型注解
class SymbolReferenceWalker extends ParseTreeWalker {
    private references: Array<{
        name: string;
        file: string;
        Line: number;
        nodeLine: number;  // 引用出现的位置
        kind: 'call' | 'variable' | 'type';
    }> = [];
    private resolvedCache = new Map<string, { file: string; Line: number } | null>();

    // 记录每个 NameNode 的定义位置，用于排除定义点
    private definitionLocations = new Set<string>();

    // 记录函数定义的 (startOffset, endOffset) 范围，用于过滤依赖
    private functionScopes = new Map<string, { start: number; end: number }>();

    constructor(
        private parseRoot: ParseNode,
        private evaluator: TypeEvaluator,
        private filePath: string,
    ) {
        super();
    }

    // 预扫描定义点
    scanDefinitions() {
        // 扫描函数定义
        const scanNode = (node: ParseNode) => {
            if (!node) return;

            // 函数定义
            if (node.nodeType === ParseNodeType.Function) {
                const funcNode = node as any;
                if (funcNode.d.name) {
                    const nameNode = funcNode.d.name;
                    this.definitionLocations.add(`${nameNode.start}:${nameNode.d.value}`);
                    // 记录函数范围: key = "函数名:起始行"
                    const funcName = nameNode.d.value;
                    const startLine = nameNode.start;
                    this.functionScopes.set(`${funcName}:${startLine}`, {
                        start: funcNode.start,
                        end: funcNode.start + funcNode.length,
                    });
                }
            }
            // 类定义
            if (node.nodeType === ParseNodeType.Class) {
                const classNode = node as any;
                if (classNode.d.name) {
                    const nameNode = classNode.d.name;
                    this.definitionLocations.add(`${nameNode.start}:${nameNode.d.value}`);
                }
            }
            // 变量赋值
            if (node.nodeType === ParseNodeType.Assignment) {
                const assignNode = node as any;
                if (assignNode.d.leftExpr?.nodeType === ParseNodeType.Name) {
                    const nameNode = assignNode.d.leftExpr;
                    this.definitionLocations.add(`${nameNode.start}:${nameNode.d.value}`);
                }
            }
            // 类型注解变量 x: int = 1
            if (node.nodeType === ParseNodeType.TypeAnnotation) {
                const typeAnnNode = node as any;
                if (typeAnnNode.d.valueExpr?.nodeType === ParseNodeType.Name) {
                    const nameNode = typeAnnNode.d.valueExpr;
                    this.definitionLocations.add(`${nameNode.start}:${nameNode.d.value}`);
                }
            }

            // 递归遍历子节点
            for (const key of Object.keys(node.d)) {
                const child = (node.d as any)[key];
                if (child) {
                    if (Array.isArray(child)) {
                        child.forEach(c => {
                            if (c && typeof c === 'object' && 'nodeType' in c) {
                                scanNode(c as ParseNode);
                            }
                        });
                    } else if (typeof child === 'object' && 'nodeType' in child) {
                        scanNode(child as ParseNode);
                    }
                }
            }
        };

        scanNode(this.parseRoot);
    }

    // 获取指定函数的范围
    getFunctionScope(funcName: string, startLine: number): { start: number; end: number } | undefined {
        return this.functionScopes.get(`${funcName}:${startLine}`);
    }

    // 处理函数/方法调用
    override visitCall(node: CallNode): boolean {
        let nameNode: NameNode | undefined;

        if (node.d.leftExpr.nodeType === ParseNodeType.Name) {
            nameNode = node.d.leftExpr;
        } else if (node.d.leftExpr.nodeType === ParseNodeType.MemberAccess) {
            nameNode = node.d.leftExpr.d.member;
        }

        if (nameNode) {
            this._resolveNameNode(nameNode, 'call', node.start);
        }
        return true;
    }

    // 处理变量引用 - 排除定义点
    override visitName(node: NameNode): boolean {
        const parent = node.parent;
        if (!parent) return true;

        // 排除函数/类定义本身
        if (parent.nodeType === ParseNodeType.Function ||
            parent.nodeType === ParseNodeType.Class ||
            parent.nodeType === ParseNodeType.Decorator) {
            return true;
        }

        // 排除 import 语句中的名称
        if (parent.nodeType === ParseNodeType.ImportAs ||
            parent.nodeType === ParseNodeType.ImportFromAs ||
            parent.nodeType === ParseNodeType.ModuleName) {
            return true;
        }

        // 排除参数 (Parameter)
        if (parent.nodeType === ParseNodeType.Parameter) {
            return true;
        }

        // 排除属性访问 (obj.attr 中的 attr)
        if (parent.nodeType === ParseNodeType.MemberAccess) {
            return true;
        }

        // 排除已记录的赋值左侧
        const key = `${node.start}:${node.d.value}`;
        if (this.definitionLocations.has(key)) {
            return true;
        }

        this._resolveNameNode(node, 'variable', node.start);
        return true;
    }

    // 处理类型注解
    override visitTypeAnnotation(node: any): boolean {
        const annotation = node.d.annotation;
        if (!annotation) return true;

        if (annotation.nodeType === ParseNodeType.Name) {
            this._resolveNameNode(annotation, 'type', annotation.start);
        } else if (annotation.nodeType === ParseNodeType.MemberAccess) {
            // 处理 typing.List[int] 等
            if (annotation.d.member) {
                this._resolveNameNode(annotation.d.member, 'type', annotation.d.member.start);
            }
        }
        return true;
    }

    // 处理成员访问 (obj.attr)
    override visitMemberAccess(node: MemberAccessNode): boolean {
        const member = node.d.member;
        const leftExpr = node.d.leftExpr;

        // 获取成员的类型
        try {
            const leftType = this.evaluator.getType(leftExpr);
            if (leftType) {
                const { doForEachSubtype } = require('../analyzer/typeUtils') as any;
                const { isClassInstance } = require('../analyzer/typeGuards') as any;
                const { lookUpObjectMember } = require('../analyzer/typeUtils') as any;

                doForEachSubtype(leftType, (subtype: any) => {
                    if (subtype && isClassInstance(subtype)) {
                        const memberInfo = lookUpObjectMember(subtype, member.d.value);
                        if (memberInfo) {
                            const decls = memberInfo.symbol.getDeclarations();
                            if (decls.length > 0) {
                                const decl = this.evaluator.resolveAliasDeclaration(decls[0], true);
                                if (decl) {
                                    this.references.push({
                                        name: member.d.value,
                                        file: decl.uri.getFilePath(),
                                        Line: decl.range.start.line,
                                        nodeLine: member.start,
                                        kind: 'variable',
                                    });
                                }
                            }
                        }
                    }
                });
            }
        } catch (e) {
            // 忽略类型解析错误
        }

        return true;
    }

    private _resolveNameNode(nameNode: NameNode, kind: 'call' | 'variable' | 'type', nodeLine: number): void {
        const nameValue = nameNode.d.value;
        if (!nameValue || nameValue === '_') return;

        const cacheKey = `${nameNode.start}:${nameValue}`;
        if (this.resolvedCache.has(cacheKey)) {
            const cached = this.resolvedCache.get(cacheKey);
            if (cached) {
                this.references.push({
                    name: nameValue,
                    file: cached.file,
                    Line: cached.Line,
                    nodeLine,
                    kind,
                });
            }
            return;
        }

        try {
            const declInfo = this.evaluator.getDeclInfoForNameNode(nameNode);
            if (declInfo?.decls && declInfo.decls.length > 0) {
                for (const decl of declInfo.decls) {
                    const resolvedDecl = this.evaluator.resolveAliasDeclaration(decl, true);
                    if (resolvedDecl &&
                        (resolvedDecl.type === DeclarationType.Function ||
                         resolvedDecl.type === DeclarationType.Class ||
                         resolvedDecl.type === DeclarationType.Variable)) {
                        const filePath = resolvedDecl.uri.getFilePath();
                        const Line = resolvedDecl.range.start.line;

                        this.resolvedCache.set(cacheKey, { file: filePath, Line });
                        this.references.push({
                            name: nameValue,
                            file: filePath,
                            Line,
                            nodeLine,
                            kind,
                        });
                        break;
                    }
                }
            } else {
                this.resolvedCache.set(cacheKey, null);
            }
        } catch (e) {
            this.resolvedCache.set(cacheKey, null);
        }
    }

    collect() {
        this.walk(this.parseRoot);
        return this.references;
    }
}

// 解析仓库
async function parseRepository(repoPath: string, verbose: boolean = false) {
    // 1. 初始化 ServiceProvider (按照 pyright 的初始化方式)
    // 始终使用 NullConsole 保证有 level 属性
    const output = verbose ? new StandardConsole(LogLevel.Log) : {
        log: () => {},
        error: () => {},
        warn: () => {},
        info: () => {},
        level: LogLevel.Error,  // 关键：需要 level 属性
    } as any;
    const tempFile = new RealTempFile();
    const fileSystem = createFromRealFileSystem(tempFile, output);
    const serviceProvider = createServiceProvider(fileSystem, output, tempFile);

    // 2. 创建配置
    const repoUri = Uri.file(repoPath, serviceProvider);
    const config = new ConfigOptions(repoUri);

    // 3. 创建 AnalyzerService
    const service = new AnalyzerService('python-parser', serviceProvider, {
        console: output,
        hostFactory: () => new FullAccessHost(serviceProvider),
        libraryReanalysisTimeProvider: () => 2 * 1000,
        configOptions: config,
        shouldRunAnalysis: () => true,
    } as any);

    // 4. 扫描 Python 文件
    const pythonFiles = scanPythonFiles(repoPath);

    if (verbose) console.error('Python files found:', pythonFiles.length);

    if (pythonFiles.length === 0) {
        console.error('No Python files found in:', repoPath);
        return;
    }

    // 5. 添加到 Service
    const fileUris = pythonFiles.map(f => Uri.file(f, serviceProvider));

    for (const uri of fileUris) {
        service.setFileOpened(uri, null, '', { type: 0 } as any);
    }

    // 6. 执行分析
    const program = service.test_program;
    await program.analyze();

    // 7. 收集结果
    const modName = path.basename(repoPath);
    const result: Repository = {
        id: repoPath,
        ASTVersion: 'v0.1.5',
        ToolVersion: 'v0.1.0',
        Path: repoPath,
        RepoVersion: {
            CommitHash: 'mock123',
            ParseTime: new Date().toISOString(),
        },
        Modules: {},
        Graph: {},
    } as any;

    const modules = result.Modules;
    const graph = result.Graph;

    // 预先收集 evaluator
    const evaluator = program.evaluator;

    // 第一遍: 收集所有文件的引用 (只遍历一次 AST)
    const fileDependencyMap = new Map<string, Array<{
        name: string;
        file: string;
        Line: number;
        nodeLine: number;
        kind: 'call' | 'variable' | 'type';
    }>>();

    for (const fileUri of fileUris) {
        const parseResults = program.getParseResults(fileUri);
        if (!parseResults || !evaluator) continue;

        const collector = new SymbolReferenceWalker(
            parseResults.parserOutput.parseTree,
            evaluator,
            fileUri.getFilePath(),
        );
        collector.scanDefinitions(); // 预扫描定义点
        const refs = collector.collect();
        fileDependencyMap.set(fileUri.getFilePath(), refs);
    }

    // 预加载源码，避免重复读取
    const sourceFileContents = new Map<string, string>();
    for (const fileUri of fileUris) {
        try {
            sourceFileContents.set(fileUri.getFilePath(), fs.readFileSync(fileUri.getFilePath(), 'utf-8'));
        } catch (e) {
            sourceFileContents.set(fileUri.getFilePath(), '');
        }
    }

    // 第二遍: 遍历 symbolTable 收集符号
    for (const fileUri of fileUris) {
        const sourceFileInfo = program.getSourceFileInfo(fileUri);
        if (!sourceFileInfo) continue;

        const boundSourceFile = sourceFileInfo.sourceFile;
        const symbolTable = boundSourceFile.getModuleSymbolTable();
        if (!symbolTable) continue;

        const relativePath = fileUri.getFilePath().replace(repoPath + '/', '');
        const packageName = path.basename(fileUri.getFilePath(), '.py');
        const relPkgPath = path.dirname(relativePath);
        // 绝对 pkgPath = modName/relPkgPath，例如 myproject/src/utils
        const absPkgPath = relPkgPath === '.' ? modName : `${modName}/${relPkgPath}`;
        const pkgPathKey = absPkgPath;

        // 获取文件依赖
        const fileDeps = fileDependencyMap.get(fileUri.getFilePath()) || [];

        // 确保 Module 存在
        if (!modules[modName]) {
            modules[modName] = {
                Language: 'python',
                Version: '0.1.0',
                Name: modName,
                Dir: '.',
                Packages: {},
                Dependencies: {},
                Files: {},
                LoadErrors: [],
            } as any;
        }

        const module = modules[modName];

        // 确保 Package 存在
        if (!module.Packages[pkgPathKey]) {
            module.Packages[pkgPathKey] = {
                IsMain: packageName === '__main__',
                IsTest: packageName.startsWith('test_') || packageName.endsWith('_test'),
                PkgPath: absPkgPath,
                Functions: {},
                Types: {},
                Vars: {},
            };
        }

        const pkg = module.Packages[pkgPathKey];

        // 添加文件信息
        if (!module.Files) module.Files = {};
        module.Files[relativePath] = {
            Path: relativePath,
            Package: absPkgPath,
            Imports: [],
        };

        // 遍历所有 symbols
        for (const [name, symbol] of symbolTable) {
            const declarations = symbol.getDeclarations();

            for (const decl of declarations) {
                if (decl.type === DeclarationType.Function) {
                    const funcNode = decl.node;
                    const startLine = decl.range.start.line;
                    const startOffset = funcNode?.start ?? 0;
                    const endOffset = funcNode ? funcNode.start + funcNode.length : 0;

                    // 获取函数源代码 (使用预加载的源码)
                    let content = '';
                    try {
                        if (funcNode) {
                            const fileContent = sourceFileContents.get(fileUri.getFilePath()) || '';
                            content = fileContent.substring(startOffset, endOffset);
                        }
                    } catch (e) {
                        // 忽略错误
                    }

                    // 获取函数签名
                    let signature = '';
                    if (evaluator && funcNode) {
                        try {
                            const funcType = evaluator.getTypeOfFunction(funcNode);
                            if (funcType) {
                                signature = evaluator.printType(funcType.functionType);
                            }
                        } catch (e) {
                            // 忽略错误
                        }
                    }

                    // 收集该函数的依赖 (从预收集过滤的依赖中)
                    // 使用 nodeLine (字符偏移) 是否落在函数体内 [startOffset, endOffset) 来过滤
                    const funcDeps = fileDeps.filter(d =>
                        d.nodeLine >= startOffset && d.nodeLine < endOffset
                    );

                    pkg.Functions![name] = {
                        Exported: true,
                        IsMethod: false,
                        IsInterfaceMethod: false,
                        ModPath: modName,
                        PkgPath: pkg.PkgPath,
                        Name: name,
                        File: relativePath,
                        Line: startLine + 1,
                        Content: content,
                        Signature: signature,
                    } as any;

                    // 添加到 Graph (收集依赖信息)
                    const funcKey = `${modName}?${pkg.PkgPath}#${name}`;
                    const funcDependencies = funcDeps.map(fc => {
                        // 从 fc.file 推导 PkgPath
                        const depRelativePath = fc.file.replace(repoPath + '/', '');
                        const depRelPkgPath = path.dirname(depRelativePath);
                        const depAbsPkgPath = depRelPkgPath === '.' ? modName : `${modName}/${depRelPkgPath}`;

                        return {
                            Kind: 'Dependency',
                            ModPath: modName,
                            PkgPath: depAbsPkgPath,
                            Name: fc.name,
                            Line: fc.Line,
                        };
                    });

                    graph[funcKey] = {
                        ModPath: modName,
                        PkgPath: pkg.PkgPath,
                        Name: name,
                        Type: 'FUNC',
                        References: [],
                        Dependencies: funcDependencies,
                    } as any;

                } else if (decl.type === DeclarationType.Class) {
                    const classNode = decl.node;

                    // 获取类源代码 (使用预加载的源码)
                    let content = '';
                    try {
                        if (classNode) {
                            const fileContent = sourceFileContents.get(fileUri.getFilePath()) || '';
                            const startOffset = classNode.start;
                            const endOffset = classNode.start + classNode.length;
                            content = fileContent.substring(startOffset, endOffset);
                        }
                    } catch (e) {
                        // 忽略错误
                    }

                    // 使用 pyright API 提取类成员
                    const classMembers = evaluator ? extractClassMembers(decl, evaluator) : { methods: [], vars: [] };

                    // 提取继承关系依赖
                    const classDeps: Relation[] = [];
                    if (evaluator && classNode) {
                        try {
                            const classType = evaluator.getTypeOfClass(classNode);
                            if (classType && classType.classType && classType.classType.shared.baseClasses) {
                                // 遍历所有父类
                                classType.classType.shared.baseClasses.forEach(baseClass => {
                                    // 解析父类的声明位置
                                    if (baseClass && baseClass.category === TypeCategory.Class) {
                                        const baseClassType = baseClass as ClassType;
                                        const baseDecl = baseClassType.shared.declaration;
                                        if (baseDecl) {
                                            const baseFilePath = baseDecl.uri.getFilePath();
                                            const baseLine = baseDecl.range.start.line;
                                            const baseName = baseClassType.shared.name;

                                            // 添加到依赖
                                            const baseRelativePath = baseFilePath.replace(repoPath + '/', '');
                                            const baseRelPkgPath = path.dirname(baseRelativePath);
                                            const baseAbsPkgPath = baseRelPkgPath === '.' ? modName : `${modName}/${baseRelPkgPath}`;

                                            classDeps.push({
                                                Kind: 'Inherit',
                                                ModPath: modName,
                                                PkgPath: baseAbsPkgPath,
                                                Name: baseName,
                                                Line: baseLine,
                                            });
                                        }
                                    }
                                });
                            }
                        } catch (e) {
                            // 忽略类型解析错误
                        }
                    }

                    pkg.Types![name] = {
                        Exported: true,
                        TypeKind: 'class',
                        ModPath: modName,
                        PkgPath: pkg.PkgPath,
                        Name: name,
                        File: relativePath,
                        Line: decl.range.start.line + 1,
                        Content: content,
                        // 添加类成员信息
                        Methods: classMembers.methods,
                        Vars: classMembers.vars,
                    } as any;

                    // 添加到 Graph
                    const classKey = `${modName}?${pkg.PkgPath}#${name}`;
                    graph[classKey] = {
                        ModPath: modName,
                        PkgPath: pkg.PkgPath,
                        Name: name,
                        Type: 'TYPE',
                        References: [],
                        Dependencies: classDeps,
                    } as any;

                } else if (decl.type === DeclarationType.Variable) {
                    // 使用 pyright API 区分类型别名和普通变量
                    const isTypeAlias = isTypeAliasDecl(decl);

                    if (isTypeAlias) {
                        // 类型别名添加到 Types
                        const varNode = decl.node;
                        let content = '';
                        try {
                            if (varNode) {
                                const fileContent = sourceFileContents.get(fileUri.getFilePath()) || '';
                                const startOffset = varNode.start;
                                const endOffset = varNode.start + varNode.length;
                                content = fileContent.substring(startOffset, endOffset);
                            }
                        } catch (e) {
                            // 忽略错误
                        }

                        pkg.Types![name] = {
                            Exported: true,
                            TypeKind: 'typedef',
                            ModPath: modName,
                            PkgPath: pkg.PkgPath,
                            Name: name,
                            File: relativePath,
                            Line: decl.range.start.line + 1,
                            Content: content,
                        } as any;

                        // 添加到 Graph
                        const typeKey = `${modName}?${pkg.PkgPath}#${name}`;
                        graph[typeKey] = {
                            ModPath: modName,
                            PkgPath: pkg.PkgPath,
                            Name: name,
                            Type: 'TYPE',
                            References: [],
                            Dependencies: [],
                        } as any;
                    } else {
                        // 普通变量添加到 Vars
                        const varNode = decl.node;

                        // 获取变量源代码 (使用预加载的源码)
                        let content = '';
                        try {
                            if (varNode) {
                                const fileContent = sourceFileContents.get(fileUri.getFilePath()) || '';
                                const startOffset = varNode.start;
                                const endOffset = varNode.start + varNode.length;
                                content = fileContent.substring(startOffset, endOffset);
                            }
                        } catch (e) {
                            // 忽略错误
                        }

                        // 注: 变量依赖通过反向构建 References 时填充

                        pkg.Vars![name] = {
                            IsExported: true,
                            IsConst: decl.isConstant || decl.isFinal,
                            IsPointer: false,
                            ModPath: modName,
                            PkgPath: pkg.PkgPath,
                            Name: name,
                            File: relativePath,
                            Line: decl.range.start.line + 1,
                            Content: content,
                        } as any;

                        // 添加到 Graph
                        const varKey = `${modName}?${pkg.PkgPath}#${name}`;
                        graph[varKey] = {
                            ModPath: modName,
                            PkgPath: pkg.PkgPath,
                            Name: name,
                            Type: 'VAR',
                            References: [],
                            Dependencies: [],
                        } as any;
                    }
                }
            }
        }
    }

    // 第三遍: 从 Dependencies 反向构建 References (Incoming)
    // 同时过滤 Dependencies，只保留存在于 Graph 中的节点（精确匹配）
    const allPkgPaths = new Set<string>(['']);
    for (const pkgKey of Object.keys(modules[modName]?.Packages || {})) {
        allPkgPaths.add(pkgKey);
    }

    for (const [key, node] of Object.entries(graph)) {
        if (!node.Dependencies || node.Dependencies.length === 0) continue;

        const validDeps: typeof node.Dependencies = [];

        for (const dep of node.Dependencies) {
            // 精确匹配：使用 dep 中的 PkgPath 直接查找
            const depAbsPkgPath = dep.PkgPath || modName;
            const exactKey = `${dep.ModPath || modName}?${depAbsPkgPath}#${dep.Name}`;

            // 首先尝试精确匹配
            let depNode = graph[exactKey];

            // 如果精确匹配找不到，才尝试其他 PkgPath 组合
            if (!depNode) {
                const possibleKeys = [
                    `${dep.ModPath || modName}?${modName}#${dep.Name}`,  // 根目录
                ];
                for (const pkgPath of allPkgPaths) {
                    possibleKeys.push(`${modName}?${pkgPath}#${dep.Name}`);
                }
                for (const depKey of possibleKeys) {
                    if (graph[depKey]) {
                        depNode = graph[depKey];
                        break;
                    }
                }
            }

            // 只有当目标节点存在于 Graph 中时，才保留这个依赖
            if (depNode) {
                validDeps.push(dep);

                // 同时构建 References（反向）
                if (depNode !== node) {  // 避免自引用
                    if (!depNode.References) depNode.References = [];
                    depNode.References.push({
                        Kind: 'Dependency',
                        ModPath: node.ModPath,
                        PkgPath: node.PkgPath,
                        Name: node.Name,
                        Line: dep.Line,
                    });
                }
            }
        }

        // 更新 Dependencies，过滤掉不存在的节点
        node.Dependencies = validDeps;
    }

    return result;
}

// 主函数
async function main() {
    const args = process.argv.slice(2);

    // 检测 verbose 模式
    const verbose = args.includes('-v') || args.includes('--verbose');
    const filteredArgs = args.filter(a => a !== '-v' && a !== '--verbose');

    let repoPath: string;
    if (filteredArgs[0] === 'parse') {
        repoPath = filteredArgs[1];
    } else {
        repoPath = filteredArgs[0];
    }

    if (!repoPath) {
        repoPath = path.join(__dirname, '../../e2e/mock-python');
    }

    // 转换为绝对路径
    let absoluteRepoPath = repoPath;
    if (!path.isAbsolute(repoPath)) {
        absoluteRepoPath = path.resolve(process.cwd(), repoPath);
    }

    // 如果是文件，获取其所在目录
    let targetPath = absoluteRepoPath;
    try {
        const stats = fs.statSync(absoluteRepoPath);
        if (stats.isFile() && absoluteRepoPath.endsWith('.py')) {
            targetPath = path.dirname(absoluteRepoPath);
            if (verbose) console.error('File detected, using directory:', targetPath);
        }
    } catch (e) {
        // 忽略错误
    }

    if (verbose) console.error('Resolved path:', targetPath);

    try {
        const result = await parseRepository(absoluteRepoPath, verbose);

        if (!result) {
            console.error('Result is undefined!');
            return;
        }

        if (verbose) console.error('Result modules:', Object.keys(result.Modules));

        // 写入文件: ~/.asts/-path-to-repo.json
        const homeDir = process.env.HOME || process.env.USERPROFILE || '/tmp';
        const astsDir = path.join(homeDir, '.asts');

        if (!fs.existsSync(astsDir)) {
            fs.mkdirSync(astsDir, { recursive: true });
        }

        const fileName = absoluteRepoPath.split('/').join('-').replace(/^-/, '') + '.json';
        const outputPath = path.join(astsDir, fileName);
        const tempPath = outputPath + '.tmp';

        fs.writeFileSync(tempPath, JSON.stringify(result, null, 2));
        fs.renameSync(tempPath, outputPath);

        console.log(outputPath);

    } catch (error) {
        console.error('Error:', error);
    }
}

main();
