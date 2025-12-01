import {
  SourceFile,
  ClassDeclaration,
  InterfaceDeclaration,
  TypeAliasDeclaration,
  EnumDeclaration,
  SyntaxKind,
  TypeNode,
  ClassExpression,
  Symbol,
  Node
} from 'ts-morph';
import { Type as UniType, Dependency } from '../types/uniast';
import { assignSymbolName, SymbolResolver } from '../utils/symbol-resolver';
import { PathUtils } from '../utils/path-utils';
import { TypeUtils } from '../utils/type-utils';
import { DependencyUtils } from '../utils/dependency-utils';

export class TypeParser {
  private symbolResolver: SymbolResolver;
  private pathUtils: PathUtils;
  private defaultExported: Symbol | undefined
  private dependencyUtils: DependencyUtils;

  constructor(projectRoot: string) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    this.symbolResolver = new SymbolResolver(null as any, projectRoot);
    this.pathUtils = new PathUtils(projectRoot);
    this.dependencyUtils = new DependencyUtils(this.symbolResolver, projectRoot);
  }

  parseTypes(sourceFile: SourceFile, moduleName: string, packagePath: string): Record<string, UniType> {
    const types: Record<string, UniType> = {};
    this.defaultExported = sourceFile.getDefaultExportSymbol()?.getAliasedSymbol()

    // Parse class declarations
    const classes = sourceFile.getClasses();
    for (const cls of classes) {
      try {
        const typeObj = this.parseClass(cls, moduleName, packagePath, sourceFile);
        types[typeObj.Name] = typeObj;
      } catch (error) {
        console.error('Error processing class:', cls, error);
      }
    }

    // Parse class expressions (anonymous classes)
    const classExpressions = sourceFile.getDescendantsOfKind(SyntaxKind.ClassExpression);
    for (let i = 0; i < classExpressions.length; i++) {
      const classExpr = classExpressions[i];
      try {
        const typeObj = this.parseClassExpression(classExpr, moduleName, packagePath, sourceFile, i);
        types[typeObj.Name] = typeObj;
      } catch (error) {
        console.error('Error processing class expression:', classExpr, error);
      }
    }

    // Parse interface declarations
    const interfaces = sourceFile.getInterfaces();
    for (const iface of interfaces) {
      try {
        const typeObj = this.parseInterface(iface, moduleName, packagePath, sourceFile);
        types[typeObj.Name] = typeObj;
      } catch (error) {
        console.error('Error processing interface:', iface, error);
      }
    }

    // Parse type alias declarations
    const typeAliases = sourceFile.getTypeAliases();
    for (const typeAlias of typeAliases) {
      try {
        const typeObj = this.parseTypeAlias(typeAlias, moduleName, packagePath, sourceFile);
        types[typeObj.Name] = typeObj;
      } catch (error) {
        console.error('Error processing type alias:', typeAlias, error);
      }
    }

    // Parse enum declarations
    const enums = sourceFile.getEnums();
    for (const enumDecl of enums) {
      try {
        const typeObj = this.parseEnum(enumDecl, moduleName, packagePath, sourceFile);
        types[typeObj.Name] = typeObj;
      } catch (error) {
        console.error('Error processing enum:', enumDecl, error);
      }
    }

    return types;
  }

  private parseClass(cls: ClassDeclaration, moduleName: string, packagePath: string, sourceFile: SourceFile): UniType {

    const sym = cls.getSymbol();
    let name = cls.getName() || 'AnonymousClass';
    if (sym) {
      name = assignSymbolName(sym);
    }
    const startLine = cls.getStartLineNumber();
    const startOffset = cls.getStart();
    const endOffset = cls.getEnd();
    const content = cls.getFullText();
    const isExported = cls.isExported() || cls.isDefaultExport() || (sym === this.defaultExported && sym !== undefined);

    // Parse methods
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const methods: Record<string, any> = {};
    const classMethods = cls.getMethods();
    for (const method of classMethods) {
      const methodName = method.getName() || 'anonymous';
      methods[methodName] = {
        ModPath: moduleName,
        PkgPath: this.getPkgPath(packagePath),
        Name: `${name}.${methodName}`
      };
    }

    // Parse implemented interfaces and extended classes
    const implementsInterfaces: Dependency[] = [];
    const extendsClasses: Dependency[] = [];

    const heritageClauses = cls.getHeritageClauses();
    for (const clause of heritageClauses) {
      const clauseType = clause.getToken();
      const typeNodes = clause.getTypeNodes();

      for (const typeNode of typeNodes) {
        const dependencies = this.extractTypeDependencies(typeNode, moduleName, packagePath);
        if (clauseType === SyntaxKind.ImplementsKeyword) {
          implementsInterfaces.push(...dependencies);
        } else if (clauseType === SyntaxKind.ExtendsKeyword) {
          extendsClasses.push(...dependencies);
        }
      }
    }

    // Combine implements and extends into Implements, but filter out external symbols
    const allImplements = [...implementsInterfaces, ...extendsClasses];

    return {
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: name,
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      Exported: isExported,
      TypeKind: 'struct',
      Content: content,
      Methods: methods,
      Implements: allImplements,
      SubStruct: [],
      InlineStruct: []
    };
  }

  private parseInterface(iface: InterfaceDeclaration, moduleName: string, packagePath: string, sourceFile: SourceFile): UniType {
    const sym = iface.getSymbol();
    let name = iface.getName() || 'AnonymousInterface';
    if (sym) {
      name = assignSymbolName(sym);
    }
    const startLine = iface.getStartLineNumber();
    const startOffset = iface.getStart();
    const endOffset = iface.getEnd();
    const content = iface.getFullText();
    const isExported = iface.isExported() || iface.isDefaultExport() || (sym === this.defaultExported && sym !== undefined);

    // Parse methods
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const methods: Record<string, any> = {};
    const interfaceMethods = iface.getMethods();
    for (const method of interfaceMethods) {
      const methodName = method.getName() || 'anonymous';
      methods[methodName] = {
        ModPath: moduleName,
        PkgPath: this.getPkgPath(packagePath),
        Name: `${name}.${methodName}`
      };
    }

    // Parse extended interfaces
    const extendsInterfaces: Dependency[] = [];
    const heritageClauses = iface.getHeritageClauses();
    for (const clause of heritageClauses) {
      if (clause.getToken() === SyntaxKind.ExtendsKeyword) {
        const typeNodes = clause.getTypeNodes();
        for (const typeNode of typeNodes) {
          const dependencies = this.extractTypeDependencies(typeNode, moduleName, packagePath);
          extendsInterfaces.push(...dependencies);
        }
      }
    }
    // Combine extends interfaces and other dependencies into Implements, but filter out external symbols
    const allImplements = [...extendsInterfaces];

    return {
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: name,
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      Exported: isExported,
      TypeKind: 'interface',
      Content: content,
      Methods: methods,
      Implements: allImplements,
      SubStruct: [],
      InlineStruct: []
    };
  }

  private parseTypeAlias(typeAlias: TypeAliasDeclaration, moduleName: string, packagePath: string, sourceFile: SourceFile): UniType {

    const sym = typeAlias.getSymbol();
    let name = typeAlias.getName() || 'AnonymousTypeAlias';
    if (sym) {
      name = assignSymbolName(sym);
    }
    const startLine = typeAlias.getStartLineNumber();
    const startOffset = typeAlias.getStart();
    const endOffset = typeAlias.getEnd();
    const content = typeAlias.getFullText();
    const isExported = typeAlias.isExported() || typeAlias.isDefaultExport() || (sym === this.defaultExported && sym !== undefined);

    // Extract type dependencies from the type alias
    const typeDependencies: Dependency[] = [];
    const typeNode = typeAlias.getTypeNode();
    if (typeNode) {
      const dependencies = this.extractTypeDependencies(typeNode, moduleName, packagePath);
      typeDependencies.push(...dependencies);
    }

    return {
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: name,
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      Exported: isExported,
      TypeKind: 'typedef',
      Content: content,
      Methods: {},
      Implements: [],
      SubStruct: [],
      InlineStruct: typeDependencies
    };
  }

  private parseEnum(enumDecl: EnumDeclaration, moduleName: string, packagePath: string, sourceFile: SourceFile): UniType {
    const sym = enumDecl.getSymbol();
    let name = enumDecl.getName() || 'AnonymousEnum';
    if (sym) {
      name = assignSymbolName(sym);
    }
    const startLine = enumDecl.getStartLineNumber();
    const startOffset = enumDecl.getStart();
    const endOffset = enumDecl.getEnd();
    const content = enumDecl.getFullText();
    const isExported = enumDecl.isExported() || enumDecl.isDefaultExport() || (sym === this.defaultExported && sym !== undefined);

    return {
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: name,
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      Exported: isExported,
      TypeKind: 'enum',
      Content: content,
      Methods: {},
      Implements: [],
      SubStruct: [],
      InlineStruct: []
    };
  }

  /**
   * Extract all type dependencies from a type expression
   * This handles union types, intersection types, generics, arrays, etc.
   * Uses SymbolResolver for consistent dependency resolution, similar to extractTypeReferences
   */
  private extractTypeDependencies(typeNode: TypeNode, moduleName: string, packagePath: string): Dependency[] {
    const dependencies: Dependency[] = [];
    const visited = new Set<string>();

    // Collect all type reference nodes (including the root typeNode itself if it's a TypeReference)
    const typeReferences: TypeNode[] = [];

    // Handle ExpressionWithTypeArguments (used in extends/implements clauses)
    if (Node.isExpressionWithTypeArguments(typeNode)) {
      const expression = typeNode.getExpression();
      let symbol: Symbol | undefined;

      if (Node.isIdentifier(expression)) {
        symbol = expression.getSymbol();
      } else if (Node.isPropertyAccessExpression(expression)) {
        symbol = expression.getSymbol();
      }

      if (symbol) {
        const [resolvedSymbol, resolvedRealSymbol] = this.symbolResolver.resolveSymbol(symbol, typeNode);
        if (resolvedSymbol && !resolvedSymbol.isExternal) {
          const decls = resolvedRealSymbol?.getDeclarations() || [];
          if (decls.length > 0) {
            const defStartOffset = decls[0].getStart();
            const defEndOffset = decls[0].getEnd();
            const key = `${resolvedSymbol.moduleName}?${resolvedSymbol.packagePath}#${resolvedSymbol.name}`;

            // Check if this is not a self-reference
            const isSelfReference = (
              resolvedSymbol.moduleName === moduleName &&
              this.getPkgPath(resolvedSymbol.packagePath || packagePath) === packagePath &&
              defStartOffset <= resolvedSymbol.startOffset &&
              resolvedSymbol.endOffset <= defEndOffset
            );

            if (!visited.has(key) && !isSelfReference) {
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

    // Handle TypeReference nodes
    if (Node.isTypeReference(typeNode)) {
      typeReferences.push(typeNode);
    }

    // Also get all descendant type references
    typeReferences.push(...typeNode.getDescendantsOfKind(SyntaxKind.TypeReference));

    // Process each type reference
    for (const typeRef of typeReferences) {
      if (!Node.isTypeReference(typeRef)) continue;

      const typeName = typeRef.getTypeName();
      let symbol: Symbol | undefined;

      if (Node.isIdentifier(typeName)) {
        symbol = typeName.getSymbol();
      } else if (Node.isQualifiedName(typeName)) {
        symbol = typeName.getRight().getSymbol();
      }

      if (!symbol) continue;

      const [resolvedSymbol, resolvedRealSymbol] = this.symbolResolver.resolveSymbol(symbol, typeRef);
      if (!resolvedSymbol || resolvedSymbol.isExternal) {
        continue;
      }

      const key = `${resolvedSymbol.moduleName}?${resolvedSymbol.packagePath}#${resolvedSymbol.name}`;
      if (visited.has(key)) {
        continue;
      }

      const decls = resolvedRealSymbol.getDeclarations();
      if (decls.length === 0) {
        continue;
      }

      const defStartOffset = decls[0].getStart();
      const defEndOffset = decls[0].getEnd();

      visited.add(key);
      const dep: Dependency = {
        ModPath: resolvedSymbol.moduleName || moduleName,
        PkgPath: this.getPkgPath(resolvedSymbol.packagePath || packagePath),
        Name: resolvedSymbol.name,
        File: resolvedSymbol.filePath,
        Line: resolvedSymbol.line,
        StartOffset: resolvedSymbol.startOffset,
        EndOffset: resolvedSymbol.endOffset
      };

      // Skip self-references
      if (
        dep.ModPath === moduleName &&
        dep.PkgPath === packagePath &&
        defStartOffset <= resolvedSymbol.startOffset &&
        resolvedSymbol.endOffset <= defEndOffset
      ) continue;
      dependencies.push(dep);
    }

    return dependencies;
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

  private parseClassExpression(classExpr: ClassExpression, moduleName: string, packagePath: string, sourceFile: SourceFile, index: number): UniType {
    const sym = classExpr.getSymbol();
    let name = `AnonymousClass_${index}`;
    if (sym) {
      const symbolName = assignSymbolName(sym);
      if (symbolName && symbolName !== 'AnonymousClass') {
        name = symbolName;
      }
    }

    const startLine = classExpr.getStartLineNumber();
    const startOffset = classExpr.getStart();
    const endOffset = classExpr.getEnd();
    const content = classExpr.getFullText();

    // Parse methods
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const methods: Record<string, any> = {};
    const classMethods = classExpr.getMethods();
    for (const method of classMethods) {
      const methodName = method.getName() || 'anonymous';
      methods[methodName] = {
        ModPath: moduleName,
        PkgPath: this.getPkgPath(packagePath),
        Name: `${name}.${methodName}`
      };
    }

    // Parse implemented interfaces and extended classes
    const implementsInterfaces: Dependency[] = [];
    const extendsClasses: Dependency[] = [];

    const heritageClauses = classExpr.getHeritageClauses();
    for (const clause of heritageClauses) {
      const clauseType = clause.getToken();
      const typeNodes = clause.getTypeNodes();

      for (const typeNode of typeNodes) {
        const dependencies = this.extractTypeDependencies(typeNode, moduleName, packagePath);

        if (clauseType === SyntaxKind.ImplementsKeyword) {
          implementsInterfaces.push(...dependencies);
        } else if (clauseType === SyntaxKind.ExtendsKeyword) {
          extendsClasses.push(...dependencies);
        }
      }
    }


    return {
      ModPath: moduleName,
      PkgPath: this.getPkgPath(packagePath),
      Name: name,
      File: this.getRelativePath(sourceFile.getFilePath()),
      Line: startLine,
      StartOffset: startOffset,
      EndOffset: endOffset,
      Exported: false, // Anonymous classes are not exported
      TypeKind: 'struct',
      Content: content,
      Methods: methods,
      Implements: [...implementsInterfaces, ...extendsClasses],
      SubStruct: [],
      InlineStruct: []
    };
  }
}