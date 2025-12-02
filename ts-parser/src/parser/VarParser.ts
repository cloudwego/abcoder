import {
  SourceFile,
  VariableDeclaration,
  PropertyDeclaration,
  EnumMember,
  Node,
  SyntaxKind,
  Project,
  Identifier,
  Symbol,
} from 'ts-morph';
import { Var as UniVar, Dependency } from '../types/uniast';
import { assignSymbolName, SymbolResolver } from '../utils/symbol-resolver';
import { PathUtils } from '../utils/path-utils';

export class VarParser {
  private symbolResolver: SymbolResolver;
  private pathUtils: PathUtils;
  private defaultExportedSym: Symbol | undefined

  constructor(project: Project, projectRoot: string) {
    this.symbolResolver = new SymbolResolver(project, projectRoot);
    this.pathUtils = new PathUtils(projectRoot);
  }

  parseVars(sourceFile: SourceFile, moduleName: string, packagePath: string): Record<string, UniVar> {
    const vars: Record<string, UniVar> = {};
    this.defaultExportedSym = sourceFile.getDefaultExportSymbol()?.getAliasedSymbol()

    // Parse variable declarations - only file-level (not inside any scope)
    const variableDeclarations = sourceFile.getVariableDeclarations();
    for (const varDecl of variableDeclarations) {
      // Skip if this variable is not at file level
      if (!this.isAtFileLevel(varDecl)) {
        continue;
      }

      // Skip if this variable declares a function (arrow function or function expression)
      const initializer = varDecl.getInitializer();
      if (initializer) {
        if (initializer.getKind() === SyntaxKind.ArrowFunction ||
          initializer.getKind() === SyntaxKind.FunctionExpression) {
          continue;
        }
      }
      try {
        const varObjs = this.parseVariableDestructuring(varDecl, moduleName, packagePath, sourceFile);
        for (const varObj of varObjs) {
          vars[varObj.Name] = varObj;
        }
      } catch (error) {
        console.error('Error processing variable:', varDecl, error);
      }
    }

    // Parse property declarations in classes - only file-level class properties
    const classes = sourceFile.getClasses();
    for (const cls of classes) {
      // Skip if this class is not at file level
      if (!this.isAtFileLevel(cls)) {
        continue;
      }

      const properties = cls.getProperties();
      for (const prop of properties) {
        // Skip if this property is not at file level
        if (!this.isAtFileLevel(prop)) {
          continue;
        }

        // Skip if this property declares a function (arrow function or function expression)
        const initializer = prop.getInitializer();
        if (initializer) {
          if (initializer.getKind() === SyntaxKind.ArrowFunction ||
            initializer.getKind() === SyntaxKind.FunctionExpression) {
            continue;
          }
        }
        try {
          const varObjs = this.parsePropertyDestructuring(prop, moduleName, packagePath, sourceFile);
          for (const varObj of varObjs) {
            vars[varObj.Name] = varObj;
          }
        } catch (error) {
          console.error('Error processing property:', prop, error);
        }

      }
    }

    // Parse enum members - only file-level enums
    const enums = sourceFile.getEnums();
    for (const enumDecl of enums) {
      // Skip if this enum is not at file level
      if (!this.isAtFileLevel(enumDecl)) {
        continue;
      }

      const members = enumDecl.getMembers();
      for (const member of members) {
        try {
          const memberObj = this.parseEnumMember(member, moduleName, packagePath, sourceFile);
          vars[memberObj.Name] = memberObj;
        } catch (error) {
          console.error('Error processing enum member:', member, error);
        }
      }
    }

    return vars;
  }

  private parseVariableDestructuring(varDecl: VariableDeclaration, moduleName: string, packagePath: string, sourceFile: SourceFile): UniVar[] {
    const results: UniVar[] = [];

    // Handle destructuring patterns
    if (varDecl.getNameNode().getKind() === SyntaxKind.ObjectBindingPattern ||
      varDecl.getNameNode().getKind() === SyntaxKind.ArrayBindingPattern) {
      return this.extractDestructuredVariables(varDecl, moduleName, packagePath, sourceFile);
    }

    // Handle regular variable declarations
    let name = varDecl.getName();
    const sym = varDecl.getSymbol();
    if (sym) {
      name = assignSymbolName(sym);
    }

    const startLine = varDecl.getStartLineNumber();
    const startOffset = varDecl.getStart();
    const endOffset = varDecl.getEnd();
    const content = varDecl.getFullText();

    const parent = varDecl.getVariableStatement();
    const isExported = parent ? (parent.isExported() || parent.isDefaultExport() || (sym === this.defaultExportedSym && sym !== undefined)) : false;
    const isConst = parent ? parent.getDeclarationKind() === 'const' : false;
    const isPointer = this.isPointerType(varDecl);

    // Parse type
    const typeNode = varDecl.getTypeNode();
    let type: Dependency | undefined;
    if (typeNode) {
      let typeSymbol: Symbol | undefined;

      // For TypeReferenceNode, get the symbol from the type name
      if (Node.isTypeReference(typeNode)) {
        const typeName = typeNode.getTypeName();
        if (Node.isIdentifier(typeName)) {
          typeSymbol = typeName.getSymbol();
        } else if (Node.isQualifiedName(typeName)) {
          typeSymbol = typeName.getRight().getSymbol();
        }
      } else {
        // For other type nodes, try to get symbol from the type itself
        typeSymbol = typeNode.getSymbol();
      }

      if (typeSymbol) {
        const [resolvedSymbol, ] = this.symbolResolver.resolveSymbol(typeSymbol, varDecl);
        if (resolvedSymbol && !resolvedSymbol.isExternal) {
          type = {
            ModPath: resolvedSymbol.moduleName || moduleName,
            PkgPath: this.getPkgPath(resolvedSymbol.packagePath || packagePath),
            Name: resolvedSymbol.name,
            File: resolvedSymbol.filePath,
            Line: resolvedSymbol.line,
            StartOffset: resolvedSymbol.startOffset,
            EndOffset: resolvedSymbol.endOffset
          };
        }
      }
    }

    // Parse dependencies from initializer
    const dependencies = this.extractInitializerDependencies(varDecl, moduleName, packagePath, sourceFile);

    results.push({
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: name,
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      IsExported: isExported,
      IsConst: isConst,
      IsPointer: isPointer,
      Content: content,
      Type: type,
      Dependencies: dependencies,
      Groups: []
    });

    return results;
  }

  private parsePropertyDestructuring(prop: PropertyDeclaration, moduleName: string, packagePath: string, sourceFile: SourceFile): UniVar[] {
    const results: UniVar[] = [];

    // Handle destructuring patterns in property declarations
    if (prop.getNameNode().getKind() === SyntaxKind.ObjectBindingPattern ||
      prop.getNameNode().getKind() === SyntaxKind.ArrayBindingPattern) {
      return this.extractDestructuredVariables(prop, moduleName, packagePath, sourceFile);
    }

    // Handle regular property declarations
    let name = prop.getName();
    const sym = prop.getSymbol();
    if (sym) {
      name = assignSymbolName(sym);
    }

    const startLine = prop.getStartLineNumber();
    const startOffset = prop.getStart();
    const endOffset = prop.getEnd();
    const content = prop.getFullText();

    const parent = prop.getParent();
    let isExported = false;
    if (Node.isClassDeclaration(parent)) {
      isExported = parent.isExported() || parent.isDefaultExport() || (parent.getSymbol() === this.defaultExportedSym && this.defaultExportedSym !== undefined);
    } else if (Node.isClassExpression(parent)) {
      // ClassExpression can be exported if assigned to an exported variable
      const grandParent = parent.getParent();
      if (Node.isVariableDeclaration(grandParent)) {
        const varStatement = grandParent.getVariableStatement();
        const varSymbol = grandParent.getSymbol();
        isExported = varStatement ? (varStatement.isExported() || varStatement.isDefaultExport() || (varSymbol === this.defaultExportedSym && varSymbol !== undefined)) : false;
      }
    }

    const isConst = false;
    const isPointer = this.isPointerType(prop);

    // Parse type
    const typeNode = prop.getTypeNode();
    let type: Dependency | undefined;
    if (typeNode) {
      let typeSymbol: Symbol | undefined;

      // For TypeReferenceNode, get the symbol from the type name
      if (Node.isTypeReference(typeNode)) {
        const typeName = typeNode.getTypeName();
        if (Node.isIdentifier(typeName)) {
          typeSymbol = typeName.getSymbol();
        } else if (Node.isQualifiedName(typeName)) {
          typeSymbol = typeName.getRight().getSymbol();
        }
      } else {
        // For other type nodes, try to get symbol from the type itself
        typeSymbol = typeNode.getSymbol();
      }

      if (typeSymbol) {
        const [resolvedSymbol, ] = this.symbolResolver.resolveSymbol(typeSymbol, prop);
        if (resolvedSymbol && !resolvedSymbol.isExternal) {
          type = {
            ModPath: resolvedSymbol.moduleName || moduleName,
            PkgPath: this.getPkgPath(resolvedSymbol.packagePath || packagePath),
            Name: resolvedSymbol.name,
            File: resolvedSymbol.filePath,
            Line: resolvedSymbol.line,
            StartOffset: resolvedSymbol.startOffset,
            EndOffset: resolvedSymbol.endOffset
          };
        }
      }
    }

    // Parse dependencies from initializer
    const dependencies = this.extractInitializerDependencies(prop, moduleName, packagePath, sourceFile);

    results.push({
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: name,
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      IsExported: isExported,
      IsConst: isConst,
      IsPointer: isPointer,
      Content: content,
      Type: type,
      Dependencies: dependencies,
      Groups: []
    });

    return results;
  }

  private parseEnumMember(member: EnumMember, moduleName: string, packagePath: string, sourceFile: SourceFile): UniVar {
    let name = member.getName();
    const sym = member.getSymbol();
    if (sym) {
      name = assignSymbolName(sym);
    }

    const startLine = member.getStartLineNumber();
    const startOffset = member.getStart();
    const endOffset = member.getEnd();
    const content = member.getFullText();

    const parent = member.getParent();
    let isExported = false;
    if (Node.isEnumDeclaration(parent)) {
      isExported = parent.isExported() || parent.isDefaultExport() || (parent.getSymbol() === this.defaultExportedSym && this.defaultExportedSym !== undefined);
    }

    const isConst = true;
    const isPointer = false;

    return {
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: name,
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      IsExported: isExported,
      IsConst: isConst,
      IsPointer: isPointer,
      Content: content,
      Type: undefined,
      Dependencies: [],
      Groups: []
    };
  }

  private extractDestructuredVariables(node: VariableDeclaration | PropertyDeclaration, moduleName: string, packagePath: string, sourceFile: SourceFile): UniVar[] {
    const results: UniVar[] = [];
    const nameNode = node.getNameNode();

    // Get the initializer for dependency extraction
    const dependencies = this.extractInitializerDependencies(node, moduleName, packagePath, sourceFile);

    // Recursively extract all identifiers from destructuring patterns
    this.extractIdentifiersFromPattern(nameNode, node, moduleName, packagePath, sourceFile, dependencies, results, node);

    return results;
  }

  private extractIdentifiersFromPattern(
    patternNode: Node,
    parentNode: VariableDeclaration | PropertyDeclaration,
    moduleName: string,
    packagePath: string,
    sourceFile: SourceFile,
    dependencies: Dependency[],
    results: UniVar[],
    originNode: Node
  ): void {
    if (Node.isIdentifier(patternNode)) {
      // Simple identifier
      const uniVar = this.createUniVar(originNode.getText(), patternNode, parentNode, moduleName, packagePath, sourceFile, dependencies)
      uniVar && results.push(uniVar);
    } else if (Node.isBindingElement(patternNode)) {
      // Handle binding elements
      const nameNode = patternNode.getNameNode();
      if (Node.isIdentifier(nameNode)) {
        const uniVar = this.createUniVar(originNode.getText(), nameNode, parentNode, moduleName, packagePath, sourceFile, dependencies)
        uniVar && results.push(uniVar);
      } else {
        // Recursively handle nested patterns
        this.extractIdentifiersFromPattern(nameNode, parentNode, moduleName, packagePath, sourceFile, dependencies, results, originNode);
      }
    } else if (Node.isObjectBindingPattern(patternNode)) {
      // Handle object destructuring
      const elements = patternNode.getElements();
      for (const element of elements) {
        this.extractIdentifiersFromPattern(element, parentNode, moduleName, packagePath, sourceFile, dependencies, results, originNode);
      }
    } else if (Node.isArrayBindingPattern(patternNode)) {
      // Handle array destructuring
      const elements = patternNode.getElements();
      for (const element of elements) {
        this.extractIdentifiersFromPattern(element, parentNode, moduleName, packagePath, sourceFile, dependencies, results, originNode);
      }
    }
  }

  private createUniVar(
    content: string,
    element: Identifier,
    parentNode: VariableDeclaration | PropertyDeclaration,
    moduleName: string,
    packagePath: string,
    sourceFile: SourceFile,
    dependencies: Dependency[]
  ): UniVar | null {
    const startLine = element.getStartLineNumber();
    const startOffset = element.getStart();
    const endOffset = element.getEnd();

    let isExported = false;
    let isConst = false;
    const symbol = element.getSymbol()
    if (!symbol) {
      return null
    }

    if (Node.isVariableDeclaration(parentNode)) {
      const parent = parentNode.getVariableStatement();
      isExported = parent ? (parent.isExported() || parent.isDefaultExport() || (parent.getSymbol() === this.defaultExportedSym && this.defaultExportedSym !== undefined)) : false;
      isConst = parent ? parent.getDeclarationKind() === 'const' : false;
    } else if (Node.isPropertyDeclaration(parentNode)) {
      const parent = parentNode.getParent();
      if (Node.isClassDeclaration(parent)) {
        isExported = parent.isExported() || parent.isDefaultExport() || (parent.getSymbol() === this.defaultExportedSym && this.defaultExportedSym !== undefined);
      } else if (Node.isClassExpression(parent)) {
        // ClassExpression can be exported if assigned to an exported variable
        const grandParent = parent.getParent();
        if (Node.isVariableDeclaration(grandParent)) {
          const varStatement = grandParent.getVariableStatement();
          const varSymbol = grandParent.getSymbol();
          isExported = varStatement ? (varStatement.isExported() || varStatement.isDefaultExport() || (varSymbol === this.defaultExportedSym && varSymbol !== undefined)) : false;
        }
      }
    }

    return {
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: assignSymbolName(symbol),
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      IsExported: isExported,
      IsConst: isConst,
      IsPointer: this.isPointerType(element),
      Content: content,
      Type: undefined,
      Dependencies: dependencies,
      Groups: []
    };
  }

  private extractInitializerDependencies(node: VariableDeclaration | PropertyDeclaration, moduleName: string, packagePath: string, _sourceFile: SourceFile): Dependency[] {
    const dependencies: Dependency[] = [];
    const visited = new Set<string>();

    const initializer = node.getInitializer();
    if (!initializer) return dependencies;


    // Extract single symbol (skip object/array literals as they create internal __object/__array symbols)
    if (initializer.getKind() !== SyntaxKind.ObjectLiteralExpression && 
        initializer.getKind() !== SyntaxKind.ArrayLiteralExpression) {
      const sourceSymbol = initializer.getSymbol();
      if (sourceSymbol) {
        const [resolvedSymbol, resolvedRealSymbol] = this.symbolResolver.resolveSymbol(sourceSymbol, node);
        if (resolvedSymbol && !resolvedSymbol.isExternal && resolvedRealSymbol) {
          // Check if the dependency is defined outside this variable declaration
          const decls = resolvedRealSymbol.getDeclarations()
          if (decls.length > 0) {
            const defStart = decls[0].getStart();
            const defEnd = decls[0].getEnd();
            if (
              moduleName !== resolvedSymbol.moduleName ||
              packagePath !== resolvedSymbol.packagePath ||
              defEnd > node.getEnd() ||
              defStart < node.getStart()
            ) {
              const key = `${resolvedSymbol.moduleName}?${resolvedSymbol.packagePath}#${resolvedSymbol.name}`;
              if (!visited.has(key)) {
                visited.add(key);
                dependencies.push({
                  ModPath: resolvedSymbol.moduleName || moduleName,
                  PkgPath: this.getPkgPath(resolvedSymbol.packagePath || packagePath),
                  Name: resolvedSymbol.name,
                  File: resolvedSymbol.filePath,
                  Line: resolvedSymbol.line,
                  StartOffset: resolvedSymbol.startOffset,
                  EndOffset: resolvedSymbol.endOffset
                });
              }
            }
          }
        }
      }
    }

    // Extract identifiers if it's a expression
    const identifiers = initializer.getDescendantsOfKind(SyntaxKind.Identifier);
    for (const identifier of identifiers) {
      // Skip if this identifier is part of a property access (struct field access)
      const parent = identifier.getParent();
      if (parent && Node.isPropertyAccessExpression(parent) && parent.getNameNode() === identifier) {
        continue;
      }

      const symbol = identifier.getSymbol();
      if (!symbol) {
        continue;
      }

      const decls = symbol.getDeclarations()
      if (decls.length === 0) {
        continue;
      }
      const isLocal = decls.some(decl =>
        decl.getAncestors().includes(initializer)
      );

      if (isLocal) {
        continue
      }

      const [resolvedSymbol, ] = this.symbolResolver.resolveSymbol(symbol, node);
      if (resolvedSymbol && !resolvedSymbol.isExternal) {
        const key = `${resolvedSymbol.moduleName}?${resolvedSymbol.packagePath}#${resolvedSymbol.name}`;
        if (visited.has(key)) {
          continue
        }
        visited.add(key);
        const dep = {
          ModPath: resolvedSymbol.moduleName || moduleName,
          PkgPath: this.getPkgPath(resolvedSymbol.packagePath || packagePath),
          Name: resolvedSymbol.name,
          File: resolvedSymbol.filePath,
          Line: resolvedSymbol.line,
          StartOffset: resolvedSymbol.startOffset,
          EndOffset: resolvedSymbol.endOffset
        }
        dependencies.push(dep);
      }
    }

    return dependencies;
  }

  private isPointerType<T extends Node>(_: T): boolean {
    return false;
  }

  private getRelativePath(filePath: string): string {
    return this.pathUtils.getRelativePath(filePath);
  }

  private getPkgPath(packagePath: string): string {
    return this.pathUtils.getPkgPath(packagePath);
  }


  private isAtFileLevel<T extends Node>(node: T): boolean {
    let parent = node.getParent();

    while (parent) {
      const kind = parent.getKind();
      // Check if this is a file-level declaration (direct child of SourceFile)
      if (kind === SyntaxKind.SourceFile) {
        return true;
      }
      // If we hit any scoped construct, it's not file-level
      if (
        kind === SyntaxKind.FunctionDeclaration ||
        kind === SyntaxKind.FunctionExpression ||
        kind === SyntaxKind.ArrowFunction ||
        kind === SyntaxKind.MethodDeclaration ||
        kind === SyntaxKind.Constructor ||
        kind === SyntaxKind.GetAccessor ||
        kind === SyntaxKind.SetAccessor ||
        kind === SyntaxKind.ClassDeclaration ||
        kind === SyntaxKind.InterfaceDeclaration ||
        kind === SyntaxKind.EnumDeclaration ||
        kind === SyntaxKind.ModuleDeclaration
      ) {
        return false;
      }

      parent = parent.getParent();
    }

    return false;
  }
}