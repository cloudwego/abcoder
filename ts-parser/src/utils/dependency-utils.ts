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
  extractAtomicTypeReferences(typeNode: TypeNode): Type[] {

    const type = typeNode.getType();
    const results: Type[] = [];
    const visited = new Set<Type>();
    let avoidUnlimitedRecursion = 0;
    
    function visit(t: Type) {
      try {
        // avoid unlimited recursion
        if(avoidUnlimitedRecursion > 1000) {
          return;
        }
        avoidUnlimitedRecursion++;
        // Make sure it's not visited
        if(visited.has(t)) {
          return;
        }
        visited.add(t);
        // If it's a generic type parameter (e.g. T, K, V), skip it
        if (t.isTypeParameter()) {
          return;
        }
        
        if(t.isUnion()) {
          t.getUnionTypes().forEach(visit);
          return;
        }

        if (t.isIntersection()) {
          t.getIntersectionTypes().forEach(visit);
          return;
        }

        if (t.isArray()) {
          const elem = t.getArrayElementType();
          if (elem) visit(elem);
          return;
        }

        if (t.isTuple()) {
          t.getTupleElements().forEach(visit);
          return;
        }

        if (t.isObject() && (t.getSymbol()?.getFlags() ?? 0) & SymbolFlags.TypeLiteral) {
          t.getProperties().forEach(prop => {
            const propType = prop.getTypeAtLocation(typeNode);
            visit(propType);
          });
          return;
        }
        results.push(t);
      } catch (error) {
        console.error('Error processing type:', t, error);
      }
    }
    
    visit(type);
    return results;
  }
}