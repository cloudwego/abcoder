import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { GraphBuilder } from '../graph-builder';
import { Repository, Identity, Relation, Function, Type, Var } from '../../types/uniast';

describe('GraphBuilder', () => {
  describe('createNodeKey', () => {
    it('should create correct node key format', () => {
      const result = GraphBuilder.createNodeKey('module/path', 'package/path', 'functionName');
      expect(result).toBe('module/path?package/path#functionName');
    });

    it('should handle empty strings', () => {
      const result = GraphBuilder.createNodeKey('', '', '');
      expect(result).toBe('?#');
    });

    it('should handle special characters', () => {
      const result = GraphBuilder.createNodeKey('mod/path', 'pkg@1.0.0', 'func-name');
      expect(result).toBe('mod/path?pkg@1.0.0#func-name');
    });

    it('should handle paths with slashes and special characters', () => {
      const result = GraphBuilder.createNodeKey('src/utils/helper', '@scope/package', 'methodName');
      expect(result).toBe('src/utils/helper?@scope/package#methodName');
    });
  });

  describe('createRelation', () => {
    it('should create relation with correct properties', () => {
      const identity: Identity = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction'
      };

      const relation = GraphBuilder.createRelation(identity, 'Dependency');

      expect(relation).toEqual({
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction',
        Kind: 'Dependency'
      });
    });

    it('should create relation with different kinds', () => {
      const identity: Identity = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testType'
      };

      const relation = GraphBuilder.createRelation(identity, 'Implement');

      expect(relation.Kind).toBe('Implement');
    });

    it('should create relation with Group kind', () => {
      const identity: Identity = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testGroup'
      };

      const relation = GraphBuilder.createRelation(identity, 'Group');

      expect(relation.Kind).toBe('Group');
    });
  });

  describe('extractDependenciesFromFunction', () => {
    it('should extract function call dependencies', () => {
      const func: Function = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 100,
        Exported: true,
        IsMethod: false,
        IsInterfaceMethod: false,
        Content: 'function test() {}',
        FunctionCalls: [
          { ModPath: 'dep/module', PkgPath: 'dep/package', Name: 'depFunction' }
        ]
      };

      const dependencies = GraphBuilder.extractDependenciesFromFunction(func);

      expect(dependencies).toHaveLength(1);
      expect(dependencies[0]).toEqual({
        ModPath: 'dep/module',
        PkgPath: 'dep/package',
        Name: 'depFunction',
        Kind: 'Dependency'
      });
    });

    it('should extract method call dependencies', () => {
      const func: Function = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 100,
        Exported: true,
        IsMethod: false,
        IsInterfaceMethod: false,
        Content: 'function test() {}',
        MethodCalls: [
          { ModPath: 'dep/module', PkgPath: 'dep/package', Name: 'depMethod' }
        ]
      };

      const dependencies = GraphBuilder.extractDependenciesFromFunction(func);

      expect(dependencies).toHaveLength(1);
      expect(dependencies[0].Name).toBe('depMethod');
    });

    it('should extract type dependencies', () => {
      const func: Function = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 100,
        Exported: true,
        IsMethod: false,
        IsInterfaceMethod: false,
        Content: 'function test() {}',
        Types: [
          { ModPath: 'types/module', PkgPath: 'types/package', Name: 'CustomType' }
        ]
      };

      const dependencies = GraphBuilder.extractDependenciesFromFunction(func);

      expect(dependencies).toHaveLength(1);
      expect(dependencies[0].Name).toBe('CustomType');
      expect(dependencies[0].Kind).toBe('Dependency');
    });

    it('should extract global variable dependencies', () => {
      const func: Function = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 100,
        Exported: true,
        IsMethod: false,
        IsInterfaceMethod: false,
        Content: 'function test() {}',
        GlobalVars: [
          { ModPath: 'globals/module', PkgPath: 'globals/package', Name: 'globalVar' }
        ]
      };

      const dependencies = GraphBuilder.extractDependenciesFromFunction(func);

      expect(dependencies).toHaveLength(1);
      expect(dependencies[0].Name).toBe('globalVar');
      expect(dependencies[0].Kind).toBe('Dependency');
    });

    it('should extract all types of dependencies', () => {
      const func: Function = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 100,
        Exported: true,
        IsMethod: false,
        IsInterfaceMethod: false,
        Content: 'function test() {}',
        FunctionCalls: [
          { ModPath: 'dep/module', PkgPath: 'dep/package', Name: 'depFunction' }
        ],
        MethodCalls: [
          { ModPath: 'dep/module', PkgPath: 'dep/package', Name: 'depMethod' }
        ],
        Types: [
          { ModPath: 'types/module', PkgPath: 'types/package', Name: 'CustomType' }
        ],
        GlobalVars: [
          { ModPath: 'globals/module', PkgPath: 'globals/package', Name: 'globalVar' }
        ]
      };

      const dependencies = GraphBuilder.extractDependenciesFromFunction(func);

      expect(dependencies).toHaveLength(4);
      expect(dependencies.map(d => d.Name)).toContain('depFunction');
      expect(dependencies.map(d => d.Name)).toContain('depMethod');
      expect(dependencies.map(d => d.Name)).toContain('CustomType');
      expect(dependencies.map(d => d.Name)).toContain('globalVar');
    });

    it('should handle function with no dependencies', () => {
      const func: Function = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 100,
        Exported: true,
        IsMethod: false,
        IsInterfaceMethod: false,
        Content: 'function test() {}'
      };

      const dependencies = GraphBuilder.extractDependenciesFromFunction(func);

      expect(dependencies).toHaveLength(0);
    });

    it('should handle empty arrays', () => {
      const func: Function = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 100,
        Exported: true,
        IsMethod: false,
        IsInterfaceMethod: false,
        Content: 'function test() {}',
        FunctionCalls: [],
        MethodCalls: [],
        Types: [],
        GlobalVars: []
      };

      const dependencies = GraphBuilder.extractDependenciesFromFunction(func);

      expect(dependencies).toHaveLength(0);
    });
  });

  describe('extractReferencesFromFunction', () => {
    it('should extract parameter references', () => {
      const func: Function = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 100,
        Exported: true,
        IsMethod: false,
        IsInterfaceMethod: false,
        Content: 'function test() {}',
        Params: [
          { ModPath: 'param/module', PkgPath: 'param/package', Name: 'ParamType' }
        ]
      };

      const references = GraphBuilder.extractReferencesFromFunction(func);

      expect(references).toHaveLength(1);
      expect(references[0]).toEqual({
        ModPath: 'param/module',
        PkgPath: 'param/package',
        Name: 'ParamType',
        Kind: 'Dependency'
      });
    });

    it('should extract result references', () => {
      const func: Function = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 100,
        Exported: true,
        IsMethod: false,
        IsInterfaceMethod: false,
        Content: 'function test() {}',
        Results: [
          { ModPath: 'result/module', PkgPath: 'result/package', Name: 'ResultType' }
        ]
      };

      const references = GraphBuilder.extractReferencesFromFunction(func);

      expect(references).toHaveLength(1);
      expect(references[0].Name).toBe('ResultType');
      expect(references[0].Kind).toBe('Dependency');
    });

    it('should extract both parameter and result references', () => {
      const func: Function = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 100,
        Exported: true,
        IsMethod: false,
        IsInterfaceMethod: false,
        Content: 'function test() {}',
        Params: [
          { ModPath: 'param/module', PkgPath: 'param/package', Name: 'ParamType' }
        ],
        Results: [
          { ModPath: 'result/module', PkgPath: 'result/package', Name: 'ResultType' }
        ]
      };

      const references = GraphBuilder.extractReferencesFromFunction(func);

      expect(references).toHaveLength(2);
      expect(references.map(r => r.Name)).toContain('ParamType');
      expect(references.map(r => r.Name)).toContain('ResultType');
    });

    it('should handle function with no references', () => {
      const func: Function = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 100,
        Exported: true,
        IsMethod: false,
        IsInterfaceMethod: false,
        Content: 'function test() {}'
      };

      const references = GraphBuilder.extractReferencesFromFunction(func);

      expect(references).toHaveLength(0);
    });

    it('should handle empty arrays', () => {
      const func: Function = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunction',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 100,
        Exported: true,
        IsMethod: false,
        IsInterfaceMethod: false,
        Content: 'function test() {}',
        Params: [],
        Results: []
      };

      const references = GraphBuilder.extractReferencesFromFunction(func);

      expect(references).toHaveLength(0);
    });
  });

  describe('buildReverseRelationships', () => {
    it('should build reverse relationships correctly', () => {
      const repository: Repository = {
        ASTVersion: 'v0.1.3',
        id: 'test-repo',
        Modules: {},
        Graph: {
          'source?pkg#func1': {
            ModPath: 'source',
            PkgPath: 'pkg',
            Name: 'func1',
            Type: 'FUNC',
            Dependencies: [
              { ModPath: 'target', PkgPath: 'pkg', Name: 'func2', Kind: 'Dependency' }
            ]
          },
          'target?pkg#func2': {
            ModPath: 'target',
            PkgPath: 'pkg',
            Name: 'func2',
            Type: 'FUNC'
          }
        }
      };

      GraphBuilder.buildReverseRelationships(repository);

      const targetNode = repository.Graph['target?pkg#func2'];
      expect(targetNode.References).toBeDefined();
      expect(targetNode.References).toHaveLength(1);
      expect(targetNode.References![0].Name).toBe('func1');
      expect(targetNode.References![0].Kind).toBe('Dependency');
    });

    it('should handle multiple reverse relationships', () => {
      const repository: Repository = {
        ASTVersion: 'v0.1.3',
        id: 'test-repo',
        Modules: {},
        Graph: {
          'source1?pkg#func1': {
            ModPath: 'source1',
            PkgPath: 'pkg',
            Name: 'func1',
            Type: 'FUNC',
            Dependencies: [
              { ModPath: 'target', PkgPath: 'pkg', Name: 'func3', Kind: 'Dependency' }
            ]
          },
          'source2?pkg#func2': {
            ModPath: 'source2',
            PkgPath: 'pkg',
            Name: 'func2',
            Type: 'FUNC',
            Dependencies: [
              { ModPath: 'target', PkgPath: 'pkg', Name: 'func3', Kind: 'Dependency' }
            ]
          },
          'target?pkg#func3': {
            ModPath: 'target',
            PkgPath: 'pkg',
            Name: 'func3',
            Type: 'FUNC'
          }
        }
      };

      GraphBuilder.buildReverseRelationships(repository);

      const targetNode = repository.Graph['target?pkg#func3'];
      expect(targetNode.References).toBeDefined();
      expect(targetNode.References).toHaveLength(2);
      expect(targetNode.References!.map(r => r.Name)).toContain('func1');
      expect(targetNode.References!.map(r => r.Name)).toContain('func2');
    });

    it('should handle missing target nodes', () => {
      const repository: Repository = {
        ASTVersion: 'v0.1.3',
        id: 'test-repo',
        Modules: {},
        Graph: {
          'source?pkg#func1': {
            ModPath: 'source',
            PkgPath: 'pkg',
            Name: 'func1',
            Type: 'FUNC',
            Dependencies: [
              { ModPath: 'missing', PkgPath: 'pkg', Name: 'missingFunc', Kind: 'Dependency' }
            ]
          }
        }
      };

      GraphBuilder.buildReverseRelationships(repository);

      // Should not throw error and should not create references for missing nodes
      expect(Object.keys(repository.Graph)).toHaveLength(1);
    });

    it('should handle empty graph', () => {
      const repository: Repository = {
        ASTVersion: 'v0.1.3',
        id: 'empty-repo',
        Modules: {},
        Graph: {}
      };

      GraphBuilder.buildReverseRelationships(repository);
      expect(Object.keys(repository.Graph)).toHaveLength(0);
    });

    it('should handle nodes without dependencies', () => {
      const repository: Repository = {
        ASTVersion: 'v0.1.3',
        id: 'test-repo',
        Modules: {},
        Graph: {
          'standalone?pkg#func': {
            ModPath: 'standalone',
            PkgPath: 'pkg',
            Name: 'func',
            Type: 'FUNC'
          }
        }
      };

      GraphBuilder.buildReverseRelationships(repository);

      const node = repository.Graph['standalone?pkg#func'];
      expect(node.References).toBeUndefined();
    });
  });

  describe('buildGraph', () => {
    it('should handle empty repository', () => {
      const repository: Repository = {
        ASTVersion: 'v0.1.3',
        id: 'empty-repo',
        Modules: {},
        Graph: {}
      };

      GraphBuilder.buildGraph(repository);
      expect(Object.keys(repository.Graph)).toHaveLength(0);
    });

    it('should build graph from repository modules with functions', () => {
      const repository: Repository = {
        ASTVersion: 'v0.1.3',
        id: 'test-repo',
        Modules: {
          'test-module': {
            Language: '',
            Version: '1.0.0',
            Name: 'test-module',
            Dir: '/test',
            Packages: {
              'test-package': {
                IsMain: true,
                IsTest: false,
                PkgPath: 'test/package',
                Functions: {
                  'testFunc': {
                    ModPath: 'test/module',
                    PkgPath: 'test/package',
                    Name: 'testFunc',
                    File: 'test.ts',
                    Line: 1,
                    StartOffset: 0,
                    EndOffset: 100,
                    Exported: true,
                    IsMethod: false,
                    IsInterfaceMethod: false,
                    Content: 'function testFunc() {}'
                  }
                },
                Types: {},
                Vars: {}
              }
            }
          }
        },
        Graph: {}
      };

      GraphBuilder.buildGraph(repository);

      const nodeKey = 'test/module?test/package#testFunc';
      expect(repository.Graph[nodeKey]).toBeDefined();
      expect(repository.Graph[nodeKey].Type).toBe('FUNC');
      expect(repository.Graph[nodeKey].Name).toBe('testFunc');
    });

    it('should build graph with types', () => {
      const mockType: Type = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'TestType',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 50,
        Exported: true,
        TypeKind: 'interface',
        Content: 'interface TestType {}'
      };

      const repository: Repository = {
        ASTVersion: 'v0.1.3',
        id: 'test-repo',
        Modules: {
          'test-module': {
            Language: '',
            Version: '1.0.0',
            Name: 'test-module',
            Dir: '/test',
            Packages: {
              'test-package': {
                IsMain: true,
                IsTest: false,
                PkgPath: 'test/package',
                Functions: {},
                Types: {
                  'TestType': mockType
                },
                Vars: {}
              }
            }
          }
        },
        Graph: {}
      };

      GraphBuilder.buildGraph(repository);

      const nodeKey = 'test/module?test/package#TestType';
      expect(repository.Graph[nodeKey]).toBeDefined();
      expect(repository.Graph[nodeKey].Type).toBe('TYPE');
      expect(repository.Graph[nodeKey].Name).toBe('TestType');
    });

    it('should build graph with types that implement interfaces', () => {
      const mockType: Type = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'TestType',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 50,
        Exported: true,
        TypeKind: 'struct',
        Content: 'class TestType implements BaseInterface {}',
        Implements: [
          { ModPath: 'base/module', PkgPath: 'base/package', Name: 'BaseInterface' }
        ]
      };

      const repository: Repository = {
        ASTVersion: 'v0.1.3',
        id: 'test-repo',
        Modules: {
          'test-module': {
            Language: '',
            Version: '1.0.0',
            Name: 'test-module',
            Dir: '/test',
            Packages: {
              'test-package': {
                IsMain: true,
                IsTest: false,
                PkgPath: 'test/package',
                Functions: {},
                Types: {
                  'TestType': mockType
                },
                Vars: {}
              }
            }
          }
        },
        Graph: {}
      };

      GraphBuilder.buildGraph(repository);

      const nodeKey = 'test/module?test/package#TestType';
      const node = repository.Graph[nodeKey];
      expect(node).toBeDefined();
      expect(node.Type).toBe('TYPE');
      expect(node.Implements).toBeDefined();
      expect(node.Implements).toHaveLength(1);
      expect(node.Implements![0].Name).toBe('BaseInterface');
      expect(node.Implements![0].Kind).toBe('Implement');
    });

    it('should build graph with variables', () => {
      const mockVariable: Var = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testVar',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 30,
        IsExported: true,
        IsConst: true,
        IsPointer: false,
        Content: 'const testVar = "value";'
      };

      const repository: Repository = {
        ASTVersion: 'v0.1.3',
        id: 'test-repo',
        Modules: {
          'test-module': {
            Language: '',
            Version: '1.0.0',
            Name: 'test-module',
            Dir: '/test',
            Packages: {
              'test-package': {
                IsMain: true,
                IsTest: false,
                PkgPath: 'test/package',
                Functions: {},
                Types: {},
                Vars: {
                  'testVar': mockVariable
                }
              }
            }
          }
        },
        Graph: {}
      };

      GraphBuilder.buildGraph(repository);

      const nodeKey = 'test/module?test/package#testVar';
      expect(repository.Graph[nodeKey]).toBeDefined();
      expect(repository.Graph[nodeKey].Type).toBe('VAR');
      expect(repository.Graph[nodeKey].Name).toBe('testVar');
    });

    it('should build graph with variables that have dependencies', () => {
      const mockVariable: Var = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testVar',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 30,
        IsExported: true,
        IsConst: true,
        IsPointer: false,
        Content: 'const testVar = someFunction();',
        Dependencies: [
          { ModPath: 'dep/module', PkgPath: 'dep/package', Name: 'someFunction' }
        ]
      };

      const repository: Repository = {
        ASTVersion: 'v0.1.3',
        id: 'test-repo',
        Modules: {
          'test-module': {
            Language: '',
            Version: '1.0.0',
            Name: 'test-module',
            Dir: '/test',
            Packages: {
              'test-package': {
                IsMain: true,
                IsTest: false,
                PkgPath: 'test/package',
                Functions: {},
                Types: {},
                Vars: {
                  'testVar': mockVariable
                }
              }
            }
          }
        },
        Graph: {}
      };

      GraphBuilder.buildGraph(repository);

      const nodeKey = 'test/module?test/package#testVar';
      const node = repository.Graph[nodeKey];
      expect(node).toBeDefined();
      expect(node.Type).toBe('VAR');
      expect(node.Dependencies).toBeDefined();
      expect(node.Dependencies).toHaveLength(1);
      expect(node.Dependencies![0].Name).toBe('someFunction');
      expect(node.Dependencies![0].Kind).toBe('Dependency');
    });

    it('should build graph with variables that have groups', () => {
      const mockVariable: Var = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testVar',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 30,
        IsExported: true,
        IsConst: true,
        IsPointer: false,
        Content: 'const testVar = "value";',
        Groups: [
          { ModPath: 'group/module', PkgPath: 'group/package', Name: 'testGroup' }
        ]
      };

      const repository: Repository = {
        ASTVersion: 'v0.1.3',
        id: 'test-repo',
        Modules: {
          'test-module': {
            Language: '',
            Version: '1.0.0',
            Name: 'test-module',
            Dir: '/test',
            Packages: {
              'test-package': {
                IsMain: true,
                IsTest: false,
                PkgPath: 'test/package',
                Functions: {},
                Types: {},
                Vars: {
                  'testVar': mockVariable
                }
              }
            }
          }
        },
        Graph: {}
      };

      GraphBuilder.buildGraph(repository);

      const nodeKey = 'test/module?test/package#testVar';
      const node = repository.Graph[nodeKey];
      expect(node).toBeDefined();
      expect(node.Type).toBe('VAR');
      expect(node.Groups).toBeDefined();
      expect(node.Groups).toHaveLength(1);
      expect(node.Groups![0].Name).toBe('testGroup');
      expect(node.Groups![0].Kind).toBe('Group');
    });

    it('should build complete graph with all node types and relationships', () => {
      const mockFunction: Function = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testFunc',
        File: 'test.ts',
        Line: 1,
        StartOffset: 0,
        EndOffset: 100,
        Exported: true,
        IsMethod: false,
        IsInterfaceMethod: false,
        Content: 'function testFunc() {}',
        FunctionCalls: [
          { ModPath: 'dep/module', PkgPath: 'dep/package', Name: 'depFunc' }
        ]
      };

      const mockType: Type = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'TestType',
        File: 'test.ts',
        Line: 10,
        StartOffset: 200,
        EndOffset: 250,
        Exported: true,
        TypeKind: 'interface',
        Content: 'interface TestType {}'
      };

      const mockVariable: Var = {
        ModPath: 'test/module',
        PkgPath: 'test/package',
        Name: 'testVar',
        File: 'test.ts',
        Line: 20,
        StartOffset: 300,
        EndOffset: 330,
        IsExported: true,
        IsConst: true,
        IsPointer: false,
        Content: 'const testVar = "value";'
      };

      const repository: Repository = {
        ASTVersion: 'v0.1.3',
        id: 'test-repo',
        Modules: {
          'test-module': {
            Language: '',
            Version: '1.0.0',
            Name: 'test-module',
            Dir: '/test',
            Packages: {
              'test-package': {
                IsMain: true,
                IsTest: false,
                PkgPath: 'test/package',
                Functions: {
                  'testFunc': mockFunction
                },
                Types: {
                  'TestType': mockType
                },
                Vars: {
                  'testVar': mockVariable
                }
              }
            }
          }
        },
        Graph: {}
      };

      GraphBuilder.buildGraph(repository);

      // Check all nodes are created
      expect(Object.keys(repository.Graph)).toHaveLength(3);
      
      const funcKey = 'test/module?test/package#testFunc';
      const typeKey = 'test/module?test/package#TestType';
      const varKey = 'test/module?test/package#testVar';
      
      expect(repository.Graph[funcKey]).toBeDefined();
      expect(repository.Graph[funcKey].Type).toBe('FUNC');
      
      expect(repository.Graph[typeKey]).toBeDefined();
      expect(repository.Graph[typeKey].Type).toBe('TYPE');
      
      expect(repository.Graph[varKey]).toBeDefined();
      expect(repository.Graph[varKey].Type).toBe('VAR');
    });
  });
});