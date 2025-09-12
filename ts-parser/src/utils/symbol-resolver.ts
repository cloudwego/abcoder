import { Project, Symbol, Node, SyntaxKind, SymbolFlags } from 'ts-morph';
import * as path from 'path';
import * as fs from 'fs';
import * as JSON5 from 'json5';


export interface ResolvedSymbol {
  name: string;
  filePath: string;
  line: number;
  column: number;
  startOffset: number;
  endOffset: number;
  isExternal: boolean;
  moduleName?: string;
  packagePath?: string;
}

export class SymbolResolver {
  private project: Project;
  private projectRoot: string;
  private resolutionCache = new Map<string, ResolvedSymbol | null>();
  private resolutionSymbolCache = new Map<string, Symbol | null>();
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private packageJsonCache = new Map<string, any>();
  private mainPackageName: string;
  private cannotResolveSymbolNames: Set<string> = new Set();

  constructor(project: Project, projectRoot: string) {
    this.project = project;
    this.projectRoot = this.normalizePath(projectRoot);
    this.cannotResolveSymbolNames = new Set();

    // Pre-cache the main package.json to avoid repeated file reads
    const mainPackageJsonPath = path.join(this.projectRoot, 'package.json');
    this.mainPackageName = 'unknown';
    try {
      if (fs.existsSync(mainPackageJsonPath)) {
        const content = fs.readFileSync(mainPackageJsonPath, 'utf8');
        const packageJson = JSON5.parse(content);
        this.mainPackageName = packageJson.name || 'unknown';
        this.packageJsonCache.set(this.projectRoot, packageJson);
      }
    } catch (e) {
      // ignore
      console.warn("Failed during parsing package.json: " + e)
    }
  }

  /**
   * Resolve a symbol to its actual definition point, following imports and exports.
   */
  resolveSymbol<T extends Node>(symbol: Symbol, expression: T): [ResolvedSymbol, Symbol] | [null, null] {
    const declarations = symbol.getDeclarations();
    if (!declarations || declarations.length === 0) {
      return [null, null];
    }
    const cacheKey = `${declarations[0].getSourceFile().getFilePath()}#${symbol.getEscapedName()}_${symbol.getDeclarations()?.[0].getStart()}`;
    if (this.resolutionCache.has(cacheKey) && this.resolutionSymbolCache.has(cacheKey)) {
      return [this.resolutionCache.get(cacheKey)!, this.resolutionSymbolCache.get(cacheKey)!];
    }

    const definitionNode = this.findActualDefinition(symbol);
    const definitionSymbol = definitionNode?.getSymbol();

    if (!definitionNode || !definitionSymbol) {
      // Log unresolved symbols only once
      if (!this.cannotResolveSymbolNames.has(symbol.getName())) {
        this.cannotResolveSymbolNames.add(symbol.getName());
        console.warn(`Symbol not found: ${symbol.getName()}.`)
      }
      this.resolutionSymbolCache.set(cacheKey, null);
      this.resolutionCache.set(cacheKey, null);
      return [null, null];
    }

    const sourceFile = definitionNode.getSourceFile();
    const filePath = sourceFile.getFilePath();
    const isExternal = sourceFile.isInNodeModules();
    
    const moduleInfo = this.extractModuleInfo(filePath, isExternal);
    const packageInfo = this.extractPackageInfo(filePath, isExternal);

    const exprSourceFile = expression.getSourceFile().getFilePath();

    const resolved: ResolvedSymbol = {
      name: assignSymbolName(definitionSymbol), // Use the original symbol name
      filePath: this.getRelativePath(exprSourceFile),
      line: expression.getStartLineNumber(),
      column: expression.getStartLinePos(),
      startOffset: expression.getStart(),
      endOffset: expression.getEnd(),
      isExternal,
      moduleName: moduleInfo.name,
      packagePath: packageInfo.path,
    };

    this.resolutionSymbolCache.set(cacheKey, definitionSymbol);
    this.resolutionCache.set(cacheKey, resolved);
    return [resolved, definitionSymbol];
  }

  /**
   * Finds the actual definition node for a symbol, traversing aliases.
   */
  findActualDefinition(symbol: Symbol): Node | null {
    let current: Symbol | undefined = symbol;
    const visited = new Set<Symbol>();

    let lastCurrent: Symbol | null = null;
  
    for (let i = 0; i < 50; i++) { // Utmost 50 iterations
      if (!current) return null;
      if (visited.has(current)) {
        console.warn("SymbolResolver: Circular alias detected:", current.getName());
        break;
      }
      visited.add(current);
  
      // If it's not an alias, break the loop
      if (!(current.getFlags() & SymbolFlags.Alias)) {
        break;
      }
  
      // If it's an alias, follow it
      const aliased = current.getAliasedSymbol();
      if (aliased && aliased !== current) {
        lastCurrent = current;
        current = aliased;
        continue;
      }
  
      // If it's an alias, check if it points to the default export
      const decls = current.getDeclarations();
      for (const decl of decls) {
        if (Node.isImportClause(decl) || Node.isImportSpecifier(decl) || Node.isExportSpecifier(decl)) {
          const importDecl = decl.getFirstAncestorByKind(SyntaxKind.ImportDeclaration)
                         ?? decl.getFirstAncestorByKind(SyntaxKind.ExportDeclaration);
          if (importDecl && Node.isImportDeclaration(importDecl)) {
            const sourceFile = importDecl.getModuleSpecifierSourceFile();
            if (sourceFile) {
              const defExport = sourceFile.getDefaultExportSymbol();
              if (defExport && defExport !== current) {
                current = defExport;
                continue;
              }
            }
          }
        }
      }
  
      break;
    }
  
    if (!current) return null;
  
    const declarations = current.getDeclarations();
    if (declarations.length === 0) { 
      if (lastCurrent && !this.cannotResolveSymbolNames.has(lastCurrent.getName())) {
        // Log unresolved symbols only once
        this.cannotResolveSymbolNames.add(lastCurrent.getName());
        console.warn("Can't parse: " + lastCurrent.getName(), ". Possibly this library has no .d.ts")
      }
      return null;
    } 
  
    // First priority: non-d.ts definition nodes
    const definition = declarations.find(d => this.isDefinitionNode(d) && !d.getSourceFile().isDeclarationFile());
    if (definition) return definition;
  
    // Second priority: non-d.ts any declaration
    const nonDeclFile = declarations.find(d => !d.getSourceFile().isDeclarationFile());
    if (nonDeclFile) return nonDeclFile;
  
    // Third priority: d.ts any definition node
    const anyDef = declarations.find(d => this.isDefinitionNode(d));
    if (anyDef) return anyDef;
  
    // Last fallback: any declaration
    return declarations[0];
  }
  



  /**
   * Check if a node represents an actual definition
   */
  private isDefinitionNode(node: Node): boolean {
    const kind = node.getKind();
    return kind === SyntaxKind.VariableDeclaration ||
           kind === SyntaxKind.FunctionDeclaration ||
           kind === SyntaxKind.ClassDeclaration ||
           kind === SyntaxKind.InterfaceDeclaration ||
           kind === SyntaxKind.TypeAliasDeclaration ||
           kind === SyntaxKind.EnumDeclaration ||
           kind === SyntaxKind.MethodDeclaration ||
           kind === SyntaxKind.PropertyDeclaration ||
           kind === SyntaxKind.Parameter ||
           kind === SyntaxKind.GetAccessor ||
           kind === SyntaxKind.SetAccessor;
  }

  /**
   * Extract module information from a file path
   */
  private extractModuleInfo(filePath: string, isExternal: boolean): { name: string } {
    if (isExternal) {
      // Handle TypeScript lib files
      if (filePath.includes('typescript/lib')) {
        const fileName = path.basename(filePath, '.d.ts');
        if (fileName.startsWith('lib.es')) {
          return { name: 'es' }; // Standard ECMAScript library
        }
        return { name: fileName };
      }
      
      const nodeModulesIndex = filePath.indexOf('node_modules');
      if (nodeModulesIndex === -1) {
        return { name: 'unknown' };
      }
      
      const afterNodeModules = filePath.substring(nodeModulesIndex + 'node_modules'.length + 1);
      const parts = afterNodeModules.split(path.sep);
      
      // Handle @types packages - map to actual runtime packages
      if (parts[0] === '@types') {
        if (parts[1] === 'node') {
          // For @types/node, extract the actual module name from the file path
          const fileName = path.basename(filePath, '.d.ts');
          return { name: fileName };
        } else if (parts.length > 2) {
          // For @types/some-package, map to the actual package
          return { name: parts[1] };
        } else {
          return { name: parts[1] };
        }
      } else if (parts[0].startsWith('@')) {
        return { name: `${parts[0]}/${parts[1]}` };
      } else {
        return { name: parts[0] };
      }
    } else {
      // For internal modules, first check if it's part of a monorepo package
      const packageJsonPath = this.findPackageJsonPath(path.dirname(filePath));
      if (packageJsonPath) {
        try {
          if (!this.packageJsonCache.has(packageJsonPath)) {
            const content = fs.readFileSync(packageJsonPath, 'utf8');
            const packageJson = JSON5.parse(content);
            this.packageJsonCache.set(packageJsonPath, packageJson);
          }
          const packageJson = this.packageJsonCache.get(packageJsonPath);
          return { name: packageJson.name || this.mainPackageName };
        } catch (e) {
          // Fall back to main package name if package.json parsing fails
          return { name: this.mainPackageName };
        }
      }
      // For internal modules, use the pre-cached main package name
      return { name: this.mainPackageName };
    }
  }

  /**
   * Extract package information from a file path
   */
  private extractPackageInfo(filePath: string, isExternal: boolean): { path: string } {
    if (isExternal) {
      return { path: this.extractModuleInfo(filePath, isExternal).name };
    }
    const dir = this.normalizePath(path.dirname(filePath));
    const packageJsonPath = this.findPackageJsonPath(dir);
    let packageRoot = this.projectRoot;
    if (packageJsonPath) {
      packageRoot = path.dirname(packageJsonPath);
    }
    const relativePath = path.relative(packageRoot, dir);
    return { path: relativePath === '' ? '.' : `${relativePath}` };
  }

  /**
   * Find the closest package.json file for a given file path
   * @param fileDir - The directory to start searching from
   * @param projectRoot - Project root
   */
  private findPackageJsonPath(fileDir: string, projectRoot?: string): string | null {
    let currentDir = fileDir;
    const stopAtRoot = this.normalizePath(projectRoot || this.projectRoot);
    
    while (this.normalizePath(currentDir) !== stopAtRoot && currentDir !== path.dirname(currentDir)) {
      const packageJsonPath = path.join(currentDir, 'package.json');
      if (fs.existsSync(packageJsonPath)) {
        return packageJsonPath;
      }
      currentDir = path.dirname(currentDir);
    }
    
    return null;
  }

  /**
   * Normalize file path for consistent output
   */
  public normalizePath(filePath: string): string {
    return filePath.replace(/\\/g, '/');
  }

  public getRelativePath(filePath: string): string {
    return path.relative(this.projectRoot, filePath).replace(/\\/g, '/');
  }

  /**
   * Clear the resolution cache
   */
  clearCache(): void {
    this.resolutionCache.clear();
  }
}

const symbolNameCache = new Map<string, Symbol>();

export function assignSymbolName(symbol: Symbol): string {
  const decls = symbol.getDeclarations()
  if(decls.length === 0) {
    return symbol.getName()
  }

  const declFile = decls[0].getSourceFile().getFilePath()

  let rawName = symbol.getName()

  // Handle methods, properties, constructors, and functions with proper naming
  const firstDecl = decls[0];
  
  // Handle class/interface members with parent prefix
  if(Node.isMethodDeclaration(firstDecl) || Node.isMethodSignature(firstDecl) || 
     Node.isPropertyDeclaration(firstDecl) || Node.isPropertySignature(firstDecl) ||
     Node.isConstructorDeclaration(firstDecl)) {
    const parent = firstDecl.getParent();
    if(Node.isClassDeclaration(parent) || Node.isInterfaceDeclaration(parent)) {
      const parentName = parent.getName() || 'AnonymousClass';
      rawName = parentName + "." + rawName
    }
  }
  
  // Handle functions with their actual name instead of 'default' for default exports
  if(Node.isFunctionDeclaration(firstDecl)) {
    const actualName = firstDecl.getName();
    if (actualName && rawName === 'default') {
      rawName = actualName;
    }
  }

  // Handle enum members with enum prefix
  if(Node.isEnumMember(firstDecl)) {
    const parent = firstDecl.getParent();
    if(Node.isEnumDeclaration(parent)) {
      const parentName = parent.getName() || 'AnonymousEnum';
      // Only add prefix if not already prefixed
      if (!rawName.startsWith(parentName + ".")) {
        rawName = parentName + "." + rawName
      }
    }
  }

  // Handle default export functions/classes
  if((Node.isFunctionDeclaration(firstDecl) || Node.isClassDeclaration(firstDecl)) && 
     firstDecl.isDefaultExport() && rawName === 'default') {
    // For default exports, use the actual name if available
    const actualName = firstDecl.getName();
    if (actualName) {
      rawName = actualName;
    }
  }

  const id = declFile + "#" + rawName
  if(!symbolNameCache.has(id)) {
    symbolNameCache.set(id, symbol)
    return rawName
  }

  const symbolExists = symbolNameCache.get(id)
  // make ts happy
  if(!symbolExists) {
    return rawName
  }

  const getDeclsPos = (symbol: Symbol) => {
    const declsPos = []
    for (const decl of symbol.getDeclarations()) {
      declsPos.push(decl.getStart())
    }
    declsPos.sort((a, b) => a - b)
    return declsPos
  }
  
  const arr1 = getDeclsPos(symbol)
  const arr2 = getDeclsPos(symbolExists)
  if(arr1.join(',') === arr2.join(',')) {
    return rawName
  }

  // mangled name
  return rawName + "_" + getDeclsPos(symbol).join(".")
}