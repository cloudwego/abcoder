import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { Identifier, Project, SyntaxKind } from 'ts-morph';
import { DependencyUtils } from '../dependency-utils';
import { SymbolResolver } from '../symbol-resolver';
import { createTestProject } from './test-utils';

describe('DependencyUtils', () => {
  let dependencyUtils: DependencyUtils;
  let symbolResolver: SymbolResolver;
  let project: Project;

  beforeEach(() => {
    project = new Project({
      compilerOptions: {
        target: 99,
        module: 1,
        allowJs: true,
        skipLibCheck: true
      }
    });
    symbolResolver = new SymbolResolver(project, process.cwd());
    dependencyUtils = new DependencyUtils(symbolResolver, process.cwd());
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('extractDependenciesFromIdentifiers', () => {
    it('should extract dependencies from simple variable usage', () => {
      const { sourceFile, cleanup } = createTestProject(`
        const helper = { value: 42 };
        const result = helper.value;
      `);

      try {
        const variableDeclaration = sourceFile.getVariableDeclaration('result');
        expect(variableDeclaration).toBeDefined();
        
        const dependencies = dependencyUtils.extractDependenciesFromIdentifiers(
          variableDeclaration! as unknown as Identifier,
          'test',
          '.'
        );

        expect(dependencies).toHaveLength(1);
        expect(dependencies[0].Name).toBe('helper');
        expect(dependencies[0].ModPath).toBe('ts-parser-utils-test');
      } finally {
        cleanup();
      }
    });

    it('should skip function calls and property access', () => {
      const { sourceFile, cleanup } = createTestProject(`
        function myFunction() { return 42; }
        const obj = { prop: myFunction() };
      `);

      try {
        const variableDeclaration = sourceFile.getVariableDeclaration('obj');
        const dependencies = dependencyUtils.extractDependenciesFromIdentifiers(
          variableDeclaration! as unknown as Identifier,
          'test',
          '.'
        );

        // Should not include myFunction as a dependency since it's a function call
        expect(dependencies).toHaveLength(0);
      } finally {
        cleanup();
      }
    });

    it('should extract dependencies from class usage', () => {
      const { sourceFile, cleanup } = createTestProject(`
        class MyClass {
          static staticProp = 42;
        }
        const instance = MyClass.staticProp;
      `);

      try {
        const variableDeclaration = sourceFile.getVariableDeclaration('instance');
        const dependencies = dependencyUtils.extractDependenciesFromIdentifiers(
          variableDeclaration! as unknown as Identifier,
          'test',
          '.'
        );

        expect(dependencies).toHaveLength(1);
        expect(dependencies[0].Name).toBe('MyClass');
      } finally {
        cleanup();
      }
    });

    it('should handle nested dependencies', () => {
      const { sourceFile, cleanup } = createTestProject(`
        const config = { db: { host: 'localhost' } };
        const host = config.db.host;
      `);

      try {
        const variableDeclaration = sourceFile.getVariableDeclaration('host');
        const dependencies = dependencyUtils.extractDependenciesFromIdentifiers(
          variableDeclaration! as unknown as Identifier,
          'test',
          '.'
        );

        expect(dependencies).toHaveLength(1);
        expect(dependencies[0].Name).toBe('config');
      } finally {
        cleanup();
      }
    });

    it('should deduplicate dependencies', () => {
      const { sourceFile, cleanup } = createTestProject(`
        const shared = { a: 1, b: 2 };
        const result = shared.a + shared.b;
      `);

      try {
        const variableDeclaration = sourceFile.getVariableDeclaration('result');
        const dependencies = dependencyUtils.extractDependenciesFromIdentifiers(
          variableDeclaration! as unknown as Identifier,
          'test',
          '.'
        );

        // Should only have one dependency for 'shared' despite multiple usages
        expect(dependencies).toHaveLength(1);
        expect(dependencies[0].Name).toBe('shared');
      } finally {
        cleanup();
      }
    });
  });

  describe('extractAtomicTypeReferences', () => {
    it('should extract simple type references', () => {
      const { sourceFile, cleanup } = createTestProject(`
        interface User {
          name: string;
        }
        type UserType = User;
      `);

      try {
        const identifiers = sourceFile.getDescendantsOfKind(SyntaxKind.Identifier); // Identifier
        const userIdentifier = identifiers.find(id => id.getText() === 'User');
        expect(userIdentifier).toBeDefined();

        const types = dependencyUtils.extractAtomicTypeReferences(userIdentifier as any);
        expect(types).toHaveLength(1);
      } finally {
        cleanup();
      }
    });

    it('should handle union types', () => {
      const { sourceFile, cleanup } = createTestProject(`
        interface A { a: string; }
        interface B { b: number; }
        type UnionType = A | B;
      `);

      try {
        const identifiers = sourceFile.getDescendantsOfKind(SyntaxKind.Identifier);
        const unionIdentifiers = identifiers.filter(id => 
          id.getText() === 'A' || id.getText() === 'B'
        );

        for (const identifier of unionIdentifiers) {
          if (identifier) {
            const types = dependencyUtils.extractAtomicTypeReferences(identifier as any);
            expect(types.length).toBeGreaterThan(0);
          }
        }
      } finally {
        cleanup();
      }
    });

    it('should handle array types', () => {
      const { sourceFile, cleanup } = createTestProject(`
        interface Item {
          id: number;
        }
        type ItemArray = Item[];
      `);

      try {
        const identifiers = sourceFile.getDescendantsOfKind(SyntaxKind.Identifier);
        const itemIdentifier = identifiers.find(id => id.getText() === 'Item');
        expect(itemIdentifier).toBeDefined();

        const types = dependencyUtils.extractAtomicTypeReferences(itemIdentifier as any);
        expect(types).toHaveLength(1);
      } finally {
        cleanup();
      }
    });

    it('should skip generic type parameters', () => {
      const { sourceFile, cleanup } = createTestProject(`
        interface Generic<T> {
          value: T;
        }
        type MyGeneric = Generic<string>;
      `);

      try {
        const identifiers = sourceFile.getDescendantsOfKind(SyntaxKind.Identifier);
        const tIdentifier = identifiers.find(id => id.getText() === 'T');
        expect(tIdentifier).toBeDefined();

        const types = dependencyUtils.extractAtomicTypeReferences(tIdentifier as any);
        expect(types).toHaveLength(0);
      } finally {
        cleanup();
      }
    });
  });

  describe('getPkgPath', () => {
    it('should return correct package path', () => {
      const { cleanup } = createTestProject(`
        const test = 42;
      `);

      try {
        // Test the private method through public interface
        const result = (dependencyUtils as any).getPkgPath('src/utils');
        expect(typeof result).toBe('string');
      } finally {
        cleanup();
      }
    });
  });
});