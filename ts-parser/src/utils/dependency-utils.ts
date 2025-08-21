import { Identifier, Node, SymbolFlags, SyntaxKind, Type, TypeNode, TypeReferenceNode } from 'ts-morph';
import { Dependency } from '../types/uniast';
import { SymbolResolver } from './symbol-resolver';
import { PathUtils } from './path-utils';

export class DependencyUtils {
  private symbolResolver: SymbolResolver;
  private projectRoot: string;

  constructor(symbolResolver: SymbolResolver, projectRoot: string) {
    this.symbolResolver = symbolResolver;
    this.projectRoot = projectRoot;
  }

  /**
   * Extract dependencies from identifiers in a node
   */
  extractDependenciesFromIdentifiers(
    node: any, 
    moduleName: string, 
    packagePath: string
  ): Dependency[] {
    const dependencies: Dependency[] = [];
    const visited = new Set<string>();
    
    const identifiers = node.getDescendantsOfKind(SyntaxKind.Identifier);
    
    for (const identifier of identifiers) {
      // Skip function calls, constructors, property names, etc.
      const parent = identifier.getParent();
      if (
        (Node.isCallExpression(parent) && parent.getExpression() === identifier) ||
        (Node.isNewExpression(parent) && parent.getExpression() === identifier) ||
        (Node.isPropertyAccessExpression(parent) && parent.getNameNode() === identifier) ||
        (Node.isBindingElement(parent) && parent.getPropertyNameNode() === identifier) ||
        (Node.isPropertyAssignment(parent) && parent.getNameNode() === identifier) ||
        (Node.isShorthandPropertyAssignment(parent) && parent.getNameNode() === identifier)
      ) {
        continue;
      }

      const symbol = identifier.getSymbol();
      if (symbol) {
        const resolvedSymbol = this.symbolResolver.resolveSymbol(symbol);
        if (resolvedSymbol && !resolvedSymbol.isExternal) {
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

    return dependencies;
  }

  /**
   * Extract atomic type references from complex type expressions
   */
  extractAtomicTypeReferences(typeNode: Identifier): Type[] {

    // Make sure it's a type identifier
    const symbol = typeNode.getSymbol();
    if (!symbol || (symbol.getFlags() & (SymbolFlags.Type | SymbolFlags.TypeParameter | SymbolFlags.TypeAlias | SymbolFlags.TypeLiteral))  === 0) {
      return [];
    }

    const type = typeNode.getType();
    const results: Type[] = [];
    const visited = new Set<Type>();
    
    function visit(t: Type) {
      // Make sure it's not visited
      if(visited.has(t)) {
        return;
      }
      // If it's a generic type parameter (e.g. T, K, V), skip it
      if (t.isTypeParameter()) {
        return;
      }
      visited.add(t);

      if (t.isUnion && t.isUnion()) {
        t.getUnionTypes().forEach(visit);
      } else if (t.isIntersection && t.isIntersection()) {
        t.getIntersectionTypes().forEach(visit);
      } else if (t.isArray && t.isArray()) {
        visit(t.getArrayElementTypeOrThrow());
      } else if (t.getAliasTypeArguments && t.getAliasTypeArguments().length > 0) {
        t.getAliasTypeArguments().forEach(visit);
      } else if (t.getTypeArguments && t.getTypeArguments().length > 0) {
        t.getTypeArguments().forEach(visit);
      } else {
        results.push(t);
      }
    }
    
    visit(type);
    return results;
  }

  private getPkgPath(packagePath: string): string {
    const pathUtils = new PathUtils(this.projectRoot);
    return pathUtils.getPkgPath(packagePath);
  }

}