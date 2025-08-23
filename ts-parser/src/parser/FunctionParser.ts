import {
  Project,
  SourceFile,
  FunctionDeclaration,
  MethodDeclaration,
  ConstructorDeclaration,
  ArrowFunction,
  FunctionExpression,
  MethodSignature,
  Node,
  SyntaxKind,
  ParameterDeclaration,
  PropertyAccessExpression,
  VariableDeclaration,
  Symbol,
  NewExpression,
  Identifier
} from 'ts-morph';
import { Function as UniFunction, Dependency, Receiver } from '../types/uniast';
import { assignSymbolName, SymbolResolver } from '../utils/symbol-resolver';
import { PathUtils } from '../utils/path-utils';
import { TypeUtils } from '../utils/type-utils';
import { DependencyUtils } from '../utils/dependency-utils';

export class FunctionParser {
  private project: Project;
  private symbolResolver: SymbolResolver;
  private projectRoot: string;
  private pathUtils: PathUtils;
  private dependencyUtils: DependencyUtils;
  private defaultExportSymbol: Symbol | undefined;

  constructor(project: Project, projectRoot: string) {
    this.project = project;
    this.symbolResolver = new SymbolResolver(project, projectRoot);
    this.projectRoot = projectRoot;
    this.pathUtils = new PathUtils(projectRoot);
    this.dependencyUtils = new DependencyUtils(this.symbolResolver, projectRoot);
  }

  parseFunctions(sourceFile: SourceFile, moduleName: string, packagePath: string): Record<string, UniFunction> {
    const functions: Record<string, UniFunction> = {};

    this.defaultExportSymbol = sourceFile.getDefaultExportSymbol()?.getAliasedSymbol()

    // Parse function declarations
    const functionDeclarations = sourceFile.getFunctions();
    for (const func of functionDeclarations) {
      const funcObj = this.parseFunction(func, moduleName, packagePath, sourceFile);
      functions[funcObj.Name] = funcObj;
    }

    // Parse method declarations in classes
    const classes = sourceFile.getClasses();
    for (const cls of classes) {
      let sym = cls.getSymbol();
      let className = ""
      if (sym) {
        className = assignSymbolName(sym)
      } else {
        className = "anonymous_" + cls.getStart()
      }
      const methods = cls.getMethods();

      for (const method of methods) {
        const methodObj = this.parseMethod(method, moduleName, packagePath, sourceFile, className);
        functions[methodObj.Name] = methodObj;
      }

      // Parse constructors
      const constructors = cls.getConstructors();
      for (const ctor of constructors) {
        const ctorObj = this.parseConstructor(ctor, moduleName, packagePath, sourceFile, className);
        functions[ctorObj.Name] = ctorObj;
      }

      // Parse static methods
      const staticMethods = cls.getStaticMethods();
      for (const staticMethod of staticMethods) {
        const methodObj = this.parseMethod(staticMethod, moduleName, packagePath, sourceFile, className);
        functions[methodObj.Name] = methodObj;
      }
    }

    // Parse arrow functions assigned to variables
    const variableDeclarations = sourceFile.getVariableDeclarations();
    for (const varDecl of variableDeclarations) {
      const initializer = varDecl.getInitializer();
      if (initializer && (Node.isArrowFunction(initializer) || Node.isFunctionExpression(initializer))) {
        let sym = varDecl.getSymbol()
        let funcName = ""
        if (sym) {
          funcName = assignSymbolName(sym)
        } else {
          funcName = "anonymous_" + varDecl.getStart()
        }
        const funcObj = this.parseArrowFunction(initializer, funcName, moduleName, packagePath, sourceFile, varDecl);
        functions[funcObj.Name] = funcObj;
      }
    }

    // Parse interface methods
    const interfaces = sourceFile.getInterfaces();
    for (const iface of interfaces) {
      const methods = iface.getMethods();

      for (const method of methods) {
        const methodObj = this.parseInterfaceMethod(method, moduleName, packagePath, sourceFile);
        functions[methodObj.Name] = methodObj;
      }
    }

    return functions;
  }

  private parseFunction(func: FunctionDeclaration, moduleName: string, packagePath: string, sourceFile: SourceFile): UniFunction {
    let symbol = func.getSymbol();
    let name = 'anonymous_' + func.getStart();
    if (symbol) {
      name = assignSymbolName(symbol)
    }
    const startLine = func.getStartLineNumber();
    const startOffset = func.getStart();
    const endOffset = func.getEnd();
    const content = func.getFullText();
    const signature = this.extractSignature(func);
    const isExported = func.isExported() || func.isDefaultExport() || (this.defaultExportSymbol === symbol && symbol !== undefined);

    // Parse parameters
    const params = this.parseParameters(func.getParameters(), moduleName, packagePath, sourceFile);

    // Parse return types
    const results = this.parseReturnTypes(func, moduleName, packagePath, sourceFile);

    // Parse function calls
    const functionCalls = this.extractFunctionCalls(func, moduleName, packagePath, sourceFile);
    const methodCalls = this.extractMethodCalls(func, moduleName, packagePath, sourceFile);

    // Extract type references and global variables from function body
    const types = this.extractTypeReferences(func, moduleName, packagePath, sourceFile);
    const globalVars = this.extractGlobalVarReferences(func, moduleName, packagePath, sourceFile);

    return {
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: name,
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      Exported: isExported,
      IsMethod: false,
      IsInterfaceMethod: false,
      Content: content,
      Signature: signature,
      Params: params,
      Results: results,
      FunctionCalls: functionCalls,
      MethodCalls: methodCalls,
      Types: types,
      GlobalVars: globalVars
    };
  }

  private parseMethod(method: MethodDeclaration, moduleName: string, packagePath: string, sourceFile: SourceFile, className: string): UniFunction {
    let symbol = method.getSymbol();
    let methodName = ""
    if (symbol) {
      methodName = assignSymbolName(symbol)
    } else {
      methodName = "anonymous_" + method.getStart()
    }
    const startLine = method.getStartLineNumber();
    const startOffset = method.getStart();
    const endOffset = method.getEnd();
    const content = method.getFullText();
    const signature = this.extractSignature(method);

    const parent = method.getParent();
    const parentSym = parent.getSymbol()
    let isExported = false;
    if (Node.isClassDeclaration(parent)) {
      isExported = parent.isExported() || parent.isDefaultExport() || (this.defaultExportSymbol === parentSym && parentSym !== undefined);
    }

    // Parse receiver
    const receiver: Receiver = {
      IsPointer: false,
      Type: {
        ModPath: moduleName,
        PkgPath: this.getPkgPath(packagePath),
        Name: className
      }
    };

    // Parse parameters
    const params = this.parseParameters(method.getParameters(), moduleName, packagePath, sourceFile);

    // Parse function calls
    const functionCalls = this.extractFunctionCalls(method, moduleName, packagePath, sourceFile);
    const methodCalls = this.extractMethodCalls(method, moduleName, packagePath, sourceFile);

    // Extract type references and global variables from method body
    const types = this.extractTypeReferences(method, moduleName, packagePath, sourceFile);
    const globalVars = this.extractGlobalVarReferences(method, moduleName, packagePath, sourceFile);

    return {
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: methodName,
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      Exported: isExported,
      IsMethod: true,
      IsInterfaceMethod: false,
      Content: content,
      Signature: signature,
      Receiver: receiver,
      Params: params,
      Results: [],
      FunctionCalls: functionCalls,
      MethodCalls: methodCalls,
      Types: types,
      GlobalVars: globalVars
    };
  }

  private parseInterfaceMethod(method: MethodSignature, moduleName: string, packagePath: string, sourceFile: SourceFile): UniFunction {
    let symbol = method.getSymbol();
    let methodName = ""
    if (symbol) {
      methodName = assignSymbolName(symbol)
    } else {
      methodName = "anonymous_" + method.getStart()
    }
    const startLine = method.getStartLineNumber();
    const startOffset = method.getStart();
    const endOffset = method.getEnd();
    const content = method.getFullText();
    const signature = method.getText();

    return {
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: methodName,
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      Exported: true,
      IsMethod: true,
      IsInterfaceMethod: true,
      Content: content,
      Signature: signature,
      Params: [],
      Results: [],
      FunctionCalls: [],
      MethodCalls: [],
      Types: [],
      GlobalVars: []
    };
  }

  private parseConstructor(ctor: ConstructorDeclaration, moduleName: string, packagePath: string, sourceFile: SourceFile, className: string): UniFunction {
    let symbol = ctor.getSymbol();
    let name = ""
    if (symbol) {
      name = assignSymbolName(symbol)
    } else {
      name = `${className}.constructor_` + ctor.getStart();
    }
    const startLine = ctor.getStartLineNumber();
    const startOffset = ctor.getStart();
    const endOffset = ctor.getEnd();
    const content = ctor.getFullText();
    const signature = this.extractSignature(ctor);

    const parent = ctor.getParent();
    let isExported = false;
    if (Node.isClassDeclaration(parent)) {
      const parentSym = parent.getSymbol()
      isExported = parent.isExported() || parent.isDefaultExport() || (this.defaultExportSymbol === parentSym && parentSym !== undefined);
    }

    // Parse parameters
    const params = this.parseParameters(ctor.getParameters(), moduleName, packagePath, sourceFile);

    // Extract type references and global variables from constructor body
    const types = this.extractTypeReferences(ctor, moduleName, packagePath, sourceFile);
    const globalVars = this.extractGlobalVarReferences(ctor, moduleName, packagePath, sourceFile);

    return {
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: name,
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      Exported: isExported,
      IsMethod: true,
      IsInterfaceMethod: false,
      Content: content,
      Signature: signature,
      Params: params,
      Results: [],
      FunctionCalls: [],
      MethodCalls: [],
      Types: types,
      GlobalVars: globalVars
    };
  }

  private parseArrowFunction(arrowFunc: ArrowFunction | FunctionExpression, name: string, moduleName: string, packagePath: string, sourceFile: SourceFile, varDecl: VariableDeclaration): UniFunction {


    const startLine = arrowFunc.getStartLineNumber();
    const startOffset = arrowFunc.getStart();
    const endOffset = arrowFunc.getEnd();
    const content = varDecl.getVariableStatement()?.getFullText().trim() || arrowFunc.getFullText();
    const signature = this.extractSignature(arrowFunc);

    // Parse parameters
    const params = this.parseParameters(arrowFunc.getParameters(), moduleName, packagePath, sourceFile);

    // Parse function calls
    const functionCalls = this.extractFunctionCalls(arrowFunc, moduleName, packagePath, sourceFile);
    const methodCalls = this.extractMethodCalls(arrowFunc, moduleName, packagePath, sourceFile);

    // Extract type references and global variables from arrow function body
    const types = this.extractTypeReferences(arrowFunc, moduleName, packagePath, sourceFile);
    const globalVars = this.extractGlobalVarReferences(arrowFunc, moduleName, packagePath, sourceFile);

    // Determine export status from the variable declaration
    const parent = varDecl.getVariableStatement();
    const isExported = parent ? (parent.isExported() || parent.isDefaultExport() || (this.defaultExportSymbol === arrowFunc.getSymbol() && this.defaultExportSymbol !== undefined)) : false;

    return {
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: name,
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      Exported: isExported,
      IsMethod: false,
      IsInterfaceMethod: false,
      Content: content,
      Signature: signature,
      Params: params,
      Results: [],
      FunctionCalls: functionCalls,
      MethodCalls: methodCalls,
      Types: types,
      GlobalVars: globalVars
    };
  }

  // TODO: parse parameters
  private parseParameters(parameters: ParameterDeclaration[], moduleName: string, packagePath: string, sourceFile: SourceFile): Dependency[] {
    const dependencies: Dependency[] = [];

    return dependencies;
  }

  // TODO: parse return types
  private parseReturnTypes(func: FunctionDeclaration | MethodSignature, moduleName: string, packagePath: string, sourceFile: SourceFile): Dependency[] {
    const results: Dependency[] = [];
    return results;
  }

  private extractFunctionCalls(
    node: FunctionDeclaration | MethodDeclaration | ConstructorDeclaration | ArrowFunction | FunctionExpression,
    moduleName: string,
    packagePath: string,
    sourceFile: SourceFile
  ): Dependency[] {
    const calls: Dependency[] = [];
    const visited = new Set<string>();
    const callExpressions = node.getDescendantsOfKind(SyntaxKind.CallExpression);

    for (const callExpr of callExpressions) {
      const expression = callExpr.getExpression();
      if (!Node.isIdentifier(expression)) {
        continue;
      }
      const symbol = expression.getSymbol();
      if (!symbol) {
        continue
      }

      const [resolvedSymbol, resolvedRealSymbol] = this.symbolResolver.resolveSymbol(symbol, expression);
      if (!resolvedSymbol || !resolvedRealSymbol) {
        continue;
      }

      const key = `${resolvedSymbol.moduleName}?${resolvedSymbol.packagePath}#${resolvedSymbol.name}`;
      if (visited.has(key)) {
        continue
      }

      visited.add(key);
      let dep: Dependency = {
        ModPath: resolvedSymbol.moduleName || moduleName,
        PkgPath: this.getPkgPath(resolvedSymbol.packagePath || packagePath),
        Name: resolvedSymbol.name,
        File: resolvedSymbol.filePath,
        Line: resolvedSymbol.line,
        StartOffset: resolvedSymbol.startOffset,
        EndOffset: resolvedSymbol.endOffset
      }

      // External function could not find decls.
      if(resolvedSymbol.isExternal) {
        calls.push(dep);
        continue;
      }

      const decls = resolvedRealSymbol.getDeclarations()
      if (decls.length === 0) {
        continue;
      }
      const defStartOffset = decls[0].getStart()
      const defEndOffset = decls[0].getEnd()
      if (
        dep.ModPath === moduleName &&
        dep.PkgPath === packagePath &&
        defEndOffset <= node.getEnd() &&
        defStartOffset >= node.getStart()
      ) {
        continue
      }
      calls.push(dep);
    }
    return calls;
  }

  private extractMethodCalls(
    node: FunctionDeclaration | MethodDeclaration | ConstructorDeclaration | ArrowFunction | FunctionExpression,
    moduleName: string,
    packagePath: string,
    sourceFile: SourceFile
  ): Dependency[] {
    const calls: Dependency[] = [];
    const visited = new Set<string>();

    // Extract property access expressions
    const propertyAccesses = node.getDescendantsOfKind(SyntaxKind.PropertyAccessExpression);
    const newCalls = node.getDescendantsOfKind(SyntaxKind.NewExpression);

    for (const propAccess of propertyAccesses) {
      // Check if this property access is part of a method call
      const parent = propAccess.getParent();
      if (parent && Node.isCallExpression(parent) && parent.getExpression() === propAccess) {
        this.processMethodCall(node, propAccess, moduleName, packagePath, sourceFile, calls, visited);
      }
    }

    // Deal new call expression
    for (const newCall of newCalls) {
      const expr = newCall.getExpression();
      let lastIdentifier: Identifier | undefined;

      if (Node.isIdentifier(expr)) {
        lastIdentifier = expr;
      } else if (Node.isPropertyAccessExpression(expr)) {
        lastIdentifier = expr.getNameNode(); 
      }

      if(lastIdentifier) {
        this.processNewCall(node, lastIdentifier, moduleName, packagePath, sourceFile, calls, visited);
      }
    }

    return calls;
  }

  private processNewCall(
    callerNode: FunctionDeclaration | MethodDeclaration | ConstructorDeclaration | ArrowFunction | FunctionExpression,
    newExpr: Identifier,
    moduleName: string,
    packagePath: string,
    sourceFile: SourceFile,
    calls: Dependency[],
    visited: Set<string>
  ): void {
    const methodSymbol = newExpr.getSymbol();

    if (!methodSymbol) return;

    const [resolvedSymbol, resolvedRealSymbol] = this.symbolResolver.resolveSymbol(methodSymbol, newExpr);
    if (!resolvedSymbol) return;

    // Handle method names like 'getX' -> 'getX'
    let nameFormat = resolvedSymbol.name;

    const key = `${resolvedSymbol.moduleName}?${resolvedSymbol.packagePath}#${nameFormat}`;

    if (visited.has(key)) {
      return
    }
    visited.add(key);

    let dep: Dependency = {
      ModPath: resolvedSymbol.moduleName || moduleName,
      PkgPath: this.getPkgPath(resolvedSymbol.packagePath || packagePath),
      Name: nameFormat,
      File: resolvedSymbol.filePath,
      Line: resolvedSymbol.line,
      StartOffset: resolvedSymbol.startOffset,
      EndOffset: resolvedSymbol.endOffset
    }

    if(resolvedSymbol.isExternal) {
      calls.push(dep);
      return;
    }

    const decls = resolvedRealSymbol.getDeclarations()
    if (decls.length === 0) {
      return;
    }
    const defStartOffset = decls[0].getStart()
    const defEndOffset = decls[0].getEnd()
    if (
      dep.ModPath === moduleName &&
      dep.PkgPath === packagePath &&
      defEndOffset <= callerNode.getEnd() &&
      defStartOffset >= callerNode.getStart()
    ) return

    calls.push(dep);
  }

  private processMethodCall(
    callerNode: FunctionDeclaration | MethodDeclaration | ConstructorDeclaration | ArrowFunction | FunctionExpression,
    propAccess: PropertyAccessExpression,
    moduleName: string,
    packagePath: string,
    sourceFile: SourceFile,
    calls: Dependency[],
    visited: Set<string>
  ): void {
    const methodSymbol = propAccess.getSymbol();

    if (!methodSymbol) return;

    const [resolvedSymbol, resolvedRealSymbol] = this.symbolResolver.resolveSymbol(methodSymbol, propAccess);
    if (!resolvedSymbol) return;

    // Handle method names like 'getX' -> 'getX'
    let nameFormat = resolvedSymbol.name;

    const key = `${resolvedSymbol.moduleName}?${resolvedSymbol.packagePath}#${nameFormat}`;

    if (visited.has(key)) {
      return
    }
    visited.add(key);

    let dep: Dependency = {
      ModPath: resolvedSymbol.moduleName || moduleName,
      PkgPath: this.getPkgPath(resolvedSymbol.packagePath || packagePath),
      Name: nameFormat,
      File: resolvedSymbol.filePath,
      Line: resolvedSymbol.line,
      StartOffset: resolvedSymbol.startOffset,
      EndOffset: resolvedSymbol.endOffset
    }

    if(resolvedSymbol.isExternal) {
      calls.push(dep);
      return;
    }

    const decls = resolvedRealSymbol.getDeclarations()
    if (decls.length === 0) {
      return;
    }
    const defStartOffset = decls[0].getStart()
    const defEndOffset = decls[0].getEnd()
    if (
      dep.ModPath === moduleName &&
      dep.PkgPath === packagePath &&
      defEndOffset <= callerNode.getEnd() &&
      defStartOffset >= callerNode.getStart()
    ) return

    calls.push(dep);
  }


  private extractTypeReferences(
    node: FunctionDeclaration | MethodDeclaration | ConstructorDeclaration | ArrowFunction | FunctionExpression,
    moduleName: string,
    packagePath: string,
    sourceFile: SourceFile
  ): Dependency[] {
    const types: Dependency[] = [];
    const visited = new Set<string>();

    // Extract from type references and find their definitions
    const typeNodes = node.getDescendantsOfKind(SyntaxKind.Identifier);

    for (const typeNode of typeNodes) {
      // Handle union and intersection types by extracting individual type references
      const typeReferences = this.dependencyUtils.extractAtomicTypeReferences(typeNode);

      for (const typeRef of typeReferences) {
        let typeName = typeRef.getText();
        if (this.isPrimitiveType(typeName)) {
          continue;
        }

        const symbol = typeRef.getSymbol();
        if (!symbol) {
          continue;
        }

        const [resolvedSymbol, resolvedRealSymbol] = this.symbolResolver.resolveSymbol(symbol, typeNode);
        if (!resolvedSymbol || resolvedSymbol.isExternal) {
          continue
        }

        const decls = resolvedRealSymbol.getDeclarations()
        if (decls.length === 0) {
          continue;
        }

        const defStartOffset = decls[0].getStart()
        const defEndOffset = decls[0].getEnd()

        typeName = resolvedSymbol.name
        const key = `${resolvedSymbol.moduleName}?${resolvedSymbol.packagePath}#${typeName}`;
        if (visited.has(key)) {
          continue
        }

        visited.add(key);
        let dep: Dependency = {
          ModPath: resolvedSymbol.moduleName || moduleName,
          PkgPath: this.getPkgPath(resolvedSymbol.packagePath || packagePath),
          Name: resolvedSymbol.name,
          File: resolvedSymbol.filePath,
          Line: resolvedSymbol.line,
          StartOffset: resolvedSymbol.startOffset,
          EndOffset: resolvedSymbol.endOffset
        };

        if (
          dep.ModPath === moduleName &&
          dep.PkgPath === packagePath &&
          defEndOffset <= node.getEnd() &&
          defStartOffset >= node.getStart()
        ) {
          continue;
        }
        types.push(dep);
      }
    }

    return types;
  }

  private extractGlobalVarReferences(
    node: FunctionDeclaration | MethodDeclaration | ConstructorDeclaration | ArrowFunction | FunctionExpression,
    moduleName: string,
    packagePath: string,
    sourceFile: SourceFile
  ): Dependency[] {
    const body = node.getBody();
    if (!body) return [];

    const globalVars: Dependency[] = [];
    const visited = new Set<string>();

    const identifiers = body.getDescendantsOfKind(SyntaxKind.Identifier);

    for (const identifier of identifiers) {

      // Skip function calls / constructor calls / property names / Namespace import
      const parent = identifier.getParent();
      if (
        // Function calls / constructor calls / property names
        (Node.isCallExpression(parent) && parent.getExpression() === identifier) ||
        (Node.isNewExpression(parent) && parent.getExpression() === identifier) ||

        // Global variable references
        (Node.isPropertyAccessExpression(parent) && parent.getNameNode() === identifier) ||

        // Destructuring assignment 的 key
        (Node.isBindingElement(parent) && parent.getPropertyNameNode() === identifier) ||

        // Global variable assignments
        (Node.isPropertyAssignment(parent) && parent.getNameNode() === identifier) ||
        (Node.isShorthandPropertyAssignment(parent) && parent.getNameNode() === identifier)
      ) {
        continue;
      }

      const symbol = identifier.getSymbol();
      if (!symbol) continue;

      const declarations = symbol.getDeclarations();
      if (declarations.length === 0) continue;

      // if all declarations are in the current function scope, then it's a local variable (including closure capture), skip
      const isLocalOrModule = declarations.every(d => {
        return (
          d.getFirstAncestor(a => a === node) !== undefined ||
          Node.isCatchClause(d.getParent()) ||
          Node.isNamespaceImport(d)
        );
      });
      if (isLocalOrModule) continue;

      // Skip built-in symbols
      const isBuiltIn = declarations.some(d => {
        const sf = d.getSourceFile();
        return sf.isFromExternalLibrary() || sf.isDeclarationFile() || sf.getFilePath().includes("lib.");
      });
      if (isBuiltIn) continue;

      let varName = identifier.getText();
      if (this.isPrimitiveType(varName)) continue;

      // Use symbol resolver
      const [resolvedSymbol, resolvedRealSymbol] = this.symbolResolver.resolveSymbol(symbol, identifier);
      if (!resolvedSymbol || resolvedSymbol.isExternal) {
        continue;
      }


      const decls = resolvedRealSymbol.getDeclarations()
      if (decls.length === 0) {
        continue;
      }

      const defStartOffset = decls[0].getStart()
      const defEndOffset = decls[0].getEnd()


      varName = resolvedSymbol.name
      const key = `${resolvedSymbol.moduleName}?${resolvedSymbol.packagePath}#${varName}`;
      if (visited.has(key)) {
        continue;
      }

      visited.add(key);

      let dep: Dependency = {
        ModPath: resolvedSymbol.moduleName || moduleName,
        PkgPath: this.getPkgPath(resolvedSymbol.packagePath || packagePath),
        Name: resolvedSymbol.name,
        File: resolvedSymbol.filePath,
        Line: resolvedSymbol.line,
        StartOffset: resolvedSymbol.startOffset,
        EndOffset: resolvedSymbol.endOffset,
      };
      if (
        dep.ModPath === moduleName &&
        dep.PkgPath === packagePath &&
        defEndOffset <= node.getEnd() &&
        defStartOffset >= node.getStart()
      ) {
        continue;
      }
      globalVars.push(dep);
    }
    return globalVars;
  }

  private isPrimitiveType(typeName: string): boolean {
    return TypeUtils.isPrimitiveType(typeName);
  }

  private getRelativePath(filePath: string): string {
    return this.pathUtils.getRelativePath(filePath);
  }

  private getPkgPath(packagePath: string): string {
    return this.pathUtils.getPkgPath(packagePath);
  }

  private extractSignature(
    node: FunctionDeclaration | MethodDeclaration | ConstructorDeclaration | ArrowFunction | FunctionExpression
  ): string {
    if (Node.isArrowFunction(node)) {
      const equalsGreaterThanToken = node.getEqualsGreaterThan();
      const length = equalsGreaterThanToken.getEnd() - node.getStart();
      return node.getText().substring(0, length);
    }

    const body = node.getBody();
    if (!body) {
      return node.getText(); // For abstract methods or declarations without a body
    }

    // For other function-like nodes, get text up to the body
    const length = body.getStart() - node.getStart();
    return node.getText().substring(0, length).trim();
  }
}