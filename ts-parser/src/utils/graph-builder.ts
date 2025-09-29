import { Repository, Node, Relation, Identity, Function } from '../types/uniast';

/**
 * Graph Builder Utilities - Shared utilities for building repository graphs
 */
export class GraphBuilder {
  /**
   * Create node key for graph
   */
  static createNodeKey(modPath: string, pkgPath: string, name: string): string {
    return `${modPath}?${pkgPath}#${name}`;
  }

  /**
   * Create relation object
   */
  static createRelation(identity: Identity, kind: Relation['Kind']): Relation {
    return {
      ModPath: identity.ModPath,
      PkgPath: identity.PkgPath,
      Name: identity.Name,
      Kind: kind,
    };
  }

  /**
   * Extract dependencies from function
   */
  static extractDependenciesFromFunction(func: Function): Relation[] {
    const dependencies: Relation[] = [];

    // Extract from function calls
    if (func.FunctionCalls) {
      for (const call of func.FunctionCalls) {
        dependencies.push(GraphBuilder.createRelation(call, 'Dependency'));
      }
    }

    // Extract from method calls
    if (func.MethodCalls) {
      for (const call of func.MethodCalls) {
        dependencies.push(GraphBuilder.createRelation(call, 'Dependency'));
      }
    }

    // Extract from types
    if (func.Types) {
      for (const type of func.Types) {
        dependencies.push(GraphBuilder.createRelation(type, 'Dependency'));
      }
    }

    // Extract from global variables
    if (func.GlobalVars) {
      for (const globalVar of func.GlobalVars) {
        dependencies.push(GraphBuilder.createRelation(globalVar, 'Dependency'));
      }
    }

    return dependencies;
  }

  /**
   * Extract references from function
   */
  static extractReferencesFromFunction(func: Function): Relation[] {
    const references: Relation[] = [];

    // Extract from parameters
    if (func.Params) {
      for (const param of func.Params) {
        references.push(GraphBuilder.createRelation(param, 'Dependency'));
      }
    }

    // Extract from results
    if (func.Results) {
      for (const result of func.Results) {
        references.push(GraphBuilder.createRelation(result, 'Dependency'));
      }
    }

    return references;
  }

  /**
   * Build reverse relationships
   */
  static buildReverseRelationships(repository: Repository): void {
    // Build a map of all relations to create reverse references
    const relationMap = new Map<string, Map<string, Relation[]>>();

    // Collect all relations
    for (const [nodeKey, node] of Object.entries(repository.Graph)) {
      if (node.Dependencies) {
        for (const dep of node.Dependencies) {
          const targetKey = GraphBuilder.createNodeKey(dep.ModPath, dep.PkgPath, dep.Name);
          if (!relationMap.has(targetKey)) {
            relationMap.set(targetKey, new Map());
          }
          if (!relationMap.get(targetKey)!.has(nodeKey)) {
            relationMap.get(targetKey)!.set(nodeKey, []);
          }
          relationMap.get(targetKey)!.get(nodeKey)!.push(dep);
        }
      }
    }

    // Add reverse references
    for (const [targetKey, referringNodes] of relationMap) {
      if (repository.Graph[targetKey]) {
        const references: Relation[] = [];
        for (const [sourceKey, relations] of referringNodes) {
          for (const relation of relations) {
            const sourceNode = repository.Graph[sourceKey];
            if (sourceNode) {
              references.push({
                ModPath: sourceNode.ModPath,
                PkgPath: sourceNode.PkgPath,
                Name: sourceNode.Name,
                Kind: 'Dependency',
              });
            } else {
              // Handle missing nodes with UNKNOWN type
              references.push({
                ModPath: relation.ModPath,
                PkgPath: relation.PkgPath,
                Name: relation.Name,
                Kind: 'Dependency',
              });
            }
          }
        }
        repository.Graph[targetKey].References = references;
      } else {
        // Create missing node with UNKNOWN type
        const parts = targetKey.split(/[?#]/);
        const modPath = parts[0];
        const pkgPath = parts[1];
        const name = parts[2];
        
        const missingNode: Node = {
          ModPath: modPath,
          PkgPath: pkgPath,
          Name: name,
          Type: 'UNKNOWN'
        };
        
        // Add references to the missing node
        const references: Relation[] = [];
        for (const [sourceKey, ] of referringNodes) {
          const sourceNode = repository.Graph[sourceKey];
          if (sourceNode) {
            references.push({
              ModPath: sourceNode.ModPath,
              PkgPath: sourceNode.PkgPath,
              Name: sourceNode.Name,
              Kind: 'Dependency'
            });
          }
        }
        missingNode.References = references;
        repository.Graph[targetKey] = missingNode;
      }
    }
  }

  /**
   * Build complete graph for repository
   */
  static buildGraph(repository: Repository): void {
    console.log(`Building graph for repository ${repository.id}`);

    // First pass: Create all nodes from functions, types, and variables
    for (const [, module] of Object.entries(repository.Modules)) {
      for (const [, pkg] of Object.entries(module.Packages)) {
        // Add functions to graph
        for (const [, func] of Object.entries(pkg.Functions)) {
          const nodeKey = GraphBuilder.createNodeKey(func.ModPath, func.PkgPath, func.Name);
          const node: Node = {
            ModPath: func.ModPath,
            PkgPath: func.PkgPath,
            Name: func.Name,
            Type: 'FUNC',
            Dependencies: GraphBuilder.extractDependenciesFromFunction(func),
            References: GraphBuilder.extractReferencesFromFunction(func),
          };

          repository.Graph[nodeKey] = node;
        }

        // Add types to graph
        for (const [, type] of Object.entries(pkg.Types)) {
          const nodeKey = GraphBuilder.createNodeKey(type.ModPath, type.PkgPath, type.Name);
          const node: Node = {
            ModPath: type.ModPath,
            PkgPath: type.PkgPath,
            Name: type.Name,
            Type: 'TYPE',
          };

          // Add implements relationships
          if (type.Implements && type.Implements.length > 0) {
            node.Implements = type.Implements.map(impl => GraphBuilder.createRelation(impl, 'Implement'));
          }

          repository.Graph[nodeKey] = node;
        }

        // Add variables to graph
        for (const [, variable] of Object.entries(pkg.Vars)) {
          const nodeKey = GraphBuilder.createNodeKey(variable.ModPath, variable.PkgPath, variable.Name);
          const node: Node = {
            ModPath: variable.ModPath,
            PkgPath: variable.PkgPath,
            Name: variable.Name,
            Type: 'VAR',
          };

          // Add dependencies from variable
          if (variable.Dependencies && variable.Dependencies.length > 0) {
            node.Dependencies = variable.Dependencies.map(dep =>
              GraphBuilder.createRelation(dep, 'Dependency')
            );
          }

          // Add groups from variable
          if (variable.Groups && variable.Groups.length > 0) {
            node.Groups = variable.Groups.map(group => GraphBuilder.createRelation(group, 'Group'));
          }

          repository.Graph[nodeKey] = node;
        }
      }
    }

    // Second pass: Add reverse relationships (References)
    GraphBuilder.buildReverseRelationships(repository);
  }
}