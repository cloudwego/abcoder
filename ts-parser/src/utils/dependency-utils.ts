import { Identifier, Node, SymbolFlags, SyntaxKind, Type, TypeNode, TypeReferenceNode, VariableDeclaration } from 'ts-morph';
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