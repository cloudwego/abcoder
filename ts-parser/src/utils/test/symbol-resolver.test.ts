import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { Project, SyntaxKind } from 'ts-morph';
import * as path from 'path';
import * as fs from 'fs';
import { SymbolResolver, assignSymbolName } from '../symbol-resolver';
import { createTestProject } from './test-utils';

describe('SymbolResolver', () => {
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
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('resolveSymbol', () => {
    it('should resolve simple variable declarations', () => {
      const { sourceFile, cleanup } = createTestProject(`
        const myVariable = 42;
      `);

      try {
        const variableDeclaration = sourceFile.getVariableDeclaration('myVariable');
        expect(variableDeclaration).toBeDefined();
        
        const symbol = variableDeclaration?.getSymbol();
        expect(symbol).toBeDefined();

        const resolvedSymbol = symbolResolver.resolveSymbol(symbol!, variableDeclaration!);
        expect(resolvedSymbol).toBeDefined();
        expect(resolvedSymbol?.name).toBe('myVariable');
        expect(resolvedSymbol?.isExternal).toBe(false);
      } finally {
        cleanup();
      }
    });

    it('should resolve function declarations', () => {
      const { sourceFile, cleanup } = createTestProject(`
        function myFunction(param: string): number {
          return 42;
        }
      `);

      try {
        const functionDeclaration = sourceFile.getFunction('myFunction');
        expect(functionDeclaration).toBeDefined();
        
        const symbol = functionDeclaration?.getSymbol();
        expect(symbol).toBeDefined();

        const resolvedSymbol = symbolResolver.resolveSymbol(symbol!, functionDeclaration!);
        expect(resolvedSymbol).toBeDefined();
        expect(resolvedSymbol?.name).toBe('myFunction');
      } finally {
        cleanup();
      }
    });

    it('should resolve class declarations', () => {
      const { project: testProject, sourceFile, cleanup } = createTestProject(`
        class MyClass {
          myMethod(): void {}
        }
      `);

      try {
        const classDeclaration = sourceFile.getClass('MyClass');
        expect(classDeclaration).toBeDefined();
        
        const symbol = classDeclaration?.getSymbol();
        expect(symbol).toBeDefined();

        const resolvedSymbol = symbolResolver.resolveSymbol(symbol!, classDeclaration!);
        expect(resolvedSymbol).toBeDefined();
        expect(resolvedSymbol?.name).toBe('MyClass');
      } finally {
        cleanup();
      }
    });

    it('should resolve interface declarations', () => {
      const { project: testProject, sourceFile, cleanup } = createTestProject(`
        interface MyInterface {
          prop: string;
        }
      `);

      try {
        const interfaceDeclaration = sourceFile.getInterface('MyInterface');
        expect(interfaceDeclaration).toBeDefined();
        
        const symbol = interfaceDeclaration?.getSymbol();
        expect(symbol).toBeDefined();

        const resolvedSymbol = symbolResolver.resolveSymbol(symbol!, interfaceDeclaration!);
        expect(resolvedSymbol).toBeDefined();
        expect(resolvedSymbol?.name).toBe('MyInterface');
      } finally {
        cleanup();
      }
    });

    it('should resolve type alias declarations', () => {
      const { project: testProject, sourceFile, cleanup } = createTestProject(`
        type MyType = string | number;
      `);

      try {
        const typeAlias = sourceFile.getTypeAlias('MyType');
        expect(typeAlias).toBeDefined();
        
        const symbol = typeAlias?.getSymbol();
        expect(symbol).toBeDefined();

        const resolvedSymbol = symbolResolver.resolveSymbol(symbol!, typeAlias!);
        expect(resolvedSymbol).toBeDefined();
        expect(resolvedSymbol?.name).toBe('MyType');
      } finally {
        cleanup();
      }
    });

    it('should resolve enum declarations', () => {
      const { project: testProject, sourceFile, cleanup } = createTestProject(`
        enum MyEnum {
          A,
          B
        }
      `);

      try {
        const enumDeclaration = sourceFile.getEnum('MyEnum');
        expect(enumDeclaration).toBeDefined();
        
        const symbol = enumDeclaration?.getSymbol();
        expect(symbol).toBeDefined();

        const resolvedSymbol = symbolResolver.resolveSymbol(symbol!, enumDeclaration!);
        expect(resolvedSymbol).toBeDefined();
        expect(resolvedSymbol?.name).toBe('MyEnum');
      } finally {
        cleanup();
      }
    });

    it('should resolve class methods with proper naming', () => {
      const { project: testProject, sourceFile, cleanup } = createTestProject(`
        class MyClass {
          myMethod(): void {}
        }
      `);

      try {
        const classDeclaration = sourceFile.getClass('MyClass');
        const method = classDeclaration?.getMethod('myMethod');
        expect(method).toBeDefined();
        
        const symbol = method?.getSymbol();
        expect(symbol).toBeDefined();

        const resolvedSymbol = symbolResolver.resolveSymbol(symbol!, method!);
        expect(resolvedSymbol).toBeDefined();
        expect(resolvedSymbol?.name).toBe('MyClass.myMethod');
      } finally {
        cleanup();
      }
    });

    it('should resolve enum members with proper naming', () => {
      const { project: testProject, sourceFile, cleanup } = createTestProject(`
        enum MyEnum {
          VALUE_A,
          VALUE_B
        }
      `);

      try {
        const enumDeclaration = sourceFile.getEnum('MyEnum');
        const member = enumDeclaration?.getMember('VALUE_A');
        expect(member).toBeDefined();
        
        const symbol = member?.getSymbol();
        expect(symbol).toBeDefined();

        const resolvedSymbol = symbolResolver.resolveSymbol(symbol!, member!);
        expect(resolvedSymbol).toBeDefined();
        expect(resolvedSymbol?.name).toBe('MyEnum.VALUE_A');
      } finally {
        cleanup();
      }
    });

    it('should handle imports correctly', () => {
      const tempDir = path.join(__dirname, 'temp', 'import-test');
      fs.mkdirSync(tempDir, { recursive: true });

      const moduleContent = `
        export const exportedValue = 42;
      `;
      const mainContent = `
        import { exportedValue } from './module';
        const localValue = exportedValue;
      `;

      fs.writeFileSync(path.join(tempDir, 'module.ts'), moduleContent);
      fs.writeFileSync(path.join(tempDir, 'main.ts'), mainContent);

      try {
        const testProject = new Project({
          compilerOptions: {
            target: 99,
            module: 1,
            allowJs: true,
            skipLibCheck: true
          }
        });

        testProject.addSourceFilesAtPaths(path.join(tempDir, '*.ts'));
        const sourceFile = testProject.getSourceFile('main.ts');
        expect(sourceFile).toBeDefined();

        const variableDeclaration = sourceFile?.getVariableDeclaration('localValue');
        expect(variableDeclaration).toBeDefined();

        const identifiers = sourceFile?.getDescendantsOfKind(SyntaxKind.Identifier);
        const importedIdentifier = identifiers?.find(id => id.getText() === 'exportedValue');
        expect(importedIdentifier).toBeDefined();

        const symbol = importedIdentifier?.getSymbol();
        if (symbol) {
          const resolvedSymbol = symbolResolver.resolveSymbol(symbol, importedIdentifier!);
          expect(resolvedSymbol).toBeDefined();
          expect(resolvedSymbol?.name).toBe('exportedValue');
        }
      } finally {
        if (fs.existsSync(tempDir)) {
          fs.rmSync(tempDir, { recursive: true, force: true });
        }
      }
    });

    it('should handle default exports', () => {
      const tempDir = path.join(__dirname, 'temp', 'default-export-test');
      fs.mkdirSync(tempDir, { recursive: true });

      const moduleContent = `
        const defaultExport = { value: 42 };
        export const namedExport = defaultExport;
        export default defaultExport;
      `;
      const mainContent = `
        import myDefault from './module';
        import { namedExport as namedExportAlias } from './module';
        const localValue = myDefault.value;
        const localNamedValue = namedExportAlias.value;
      `;

      fs.writeFileSync(path.join(tempDir, 'module.ts'), moduleContent);
      fs.writeFileSync(path.join(tempDir, 'main.ts'), mainContent);

      try {
        const testProject = new Project({
          compilerOptions: {
            target: 99,
            module: 1,
            allowJs: true,
            skipLibCheck: true
          }
        });

        testProject.addSourceFilesAtPaths(path.join(tempDir, '*.ts'));
        const sourceFile = testProject.getSourceFile('main.ts');
        expect(sourceFile).toBeDefined();

        const identifiers = sourceFile?.getDescendantsOfKind(SyntaxKind.Identifier);
        const importedIdentifier = identifiers?.find(id => id.getText() === 'myDefault');
        expect(importedIdentifier).toBeDefined();

        let symbol = importedIdentifier?.getSymbol();
        if (symbol) {
          const resolvedSymbol = symbolResolver.resolveSymbol(symbol, importedIdentifier!);
          expect(resolvedSymbol).toBeDefined();
          expect(resolvedSymbol?.name).toBe('defaultExport');
        }

        const namedImportIdentifier = identifiers?.find(id => id.getText() === 'namedExportAlias');
        expect(namedImportIdentifier).toBeDefined();

        symbol = namedImportIdentifier?.getSymbol();
        if (symbol) {
          const resolvedSymbol = symbolResolver.resolveSymbol(symbol, namedImportIdentifier!);
          expect(resolvedSymbol).toBeDefined();
          expect(resolvedSymbol?.name).toBe('namedExport');
        }
      } finally {
        if (fs.existsSync(tempDir)) {
          fs.rmSync(tempDir, { recursive: true, force: true });
        }
      }
    });

    it('should handle circular imports gracefully', () => {
      const tempDir = path.join(__dirname, 'temp', 'circular-test');
      fs.mkdirSync(tempDir, { recursive: true });

      const moduleAContent = `
        import { b } from './moduleB';
        export const a = 42;
      `;
      const moduleBContent = `
        import { a } from './moduleA';
        export const b = a + 1;
      `;

      fs.writeFileSync(path.join(tempDir, 'moduleA.ts'), moduleAContent);
      fs.writeFileSync(path.join(tempDir, 'moduleB.ts'), moduleBContent);

      try {
        const testProject = new Project({
          compilerOptions: {
            target: 99,
            module: 1,
            allowJs: true,
            skipLibCheck: true
          }
        });

        testProject.addSourceFilesAtPaths(path.join(tempDir, '*.ts'));
        const sourceFile = testProject.getSourceFile('moduleB.ts');
        expect(sourceFile).toBeDefined();

        const identifiers = sourceFile?.getDescendantsOfKind(SyntaxKind.Identifier);
        const importedIdentifier = identifiers?.find(id => id.getText() === 'a');
        expect(importedIdentifier).toBeDefined();

        const symbol = importedIdentifier?.getSymbol();
        if (symbol) {
          const resolvedSymbol = symbolResolver.resolveSymbol(symbol!, importedIdentifier!);
          expect(resolvedSymbol).toBeDefined();
          expect(resolvedSymbol?.name).toBe('a');
        }
      } finally {
        if (fs.existsSync(tempDir)) {
          fs.rmSync(tempDir, { recursive: true, force: true });
        }
      }
    });
  });

  describe('extractModuleInfo', () => {
    it('should extract module info for internal files', () => {
      const { project: testProject, sourceFile, cleanup } = createTestProject(`
        const test = 42;
      `);

      try {
        const filePath = sourceFile.getFilePath();
        const moduleInfo = (symbolResolver as any).extractModuleInfo(filePath, false);
        expect(moduleInfo).toBeDefined();
        expect(typeof moduleInfo.name).toBe('string');
      } finally {
        cleanup();
      }
    });
  });

  describe('extractPackageInfo', () => {
    it('should extract package info for internal files', () => {
      const { project: testProject, sourceFile, cleanup } = createTestProject(`
        const test = 42;
      `);

      try {
        const filePath = sourceFile.getFilePath();
        const packageInfo = (symbolResolver as any).extractPackageInfo(filePath, false);
        expect(packageInfo).toBeDefined();
        expect(typeof packageInfo.path).toBe('string');
      } finally {
        cleanup();
      }
    });
  });

  describe('clearCache', () => {
    it('should clear the resolution cache', () => {
      const { project: testProject, sourceFile, cleanup } = createTestProject(`
        const test = 42;
      `);

      try {
        const variableDeclaration = sourceFile.getVariableDeclaration('test');
        const symbol = variableDeclaration?.getSymbol();
        
        if (symbol) {
          // First resolution
          const resolved1 = symbolResolver.resolveSymbol(symbol!, variableDeclaration!);
          expect(resolved1).toBeDefined();

          // Clear cache
          symbolResolver.clearCache();

          // Second resolution should work the same
          const resolved2 = symbolResolver.resolveSymbol(symbol!, variableDeclaration!);
          expect(resolved2).toBeDefined();
          expect(resolved2?.name).toBe(resolved1?.name);
        }
      } finally {
        cleanup();
      }
    });
  });

  describe('assignSymbolName', () => {
    it('should handle simple variable names', () => {
      const { project: testProject, sourceFile, cleanup } = createTestProject(`
        const simpleVar = 42;
      `);

      try {
        const variableDeclaration = sourceFile.getVariableDeclaration('simpleVar');
        const symbol = variableDeclaration?.getSymbol();
        expect(symbol).toBeDefined();

        if (symbol) {
          const name = assignSymbolName(symbol);
          expect(name).toBe('simpleVar');
        }
      } finally {
        cleanup();
      }
    });

    it('should handle class methods with parent prefix', () => {
      const { project: testProject, sourceFile, cleanup } = createTestProject(`
        class MyClass {
          myMethod(): void {}
        }
      `);

      try {
        const classDeclaration = sourceFile.getClass('MyClass');
        const method = classDeclaration?.getMethod('myMethod');
        const symbol = method?.getSymbol();
        expect(symbol).toBeDefined();

        if (symbol) {
          const name = assignSymbolName(symbol);
          expect(name).toBe('MyClass.myMethod');
        }
      } finally {
        cleanup();
      }
    });

    it('should handle enum members with enum prefix', () => {
      const { project: testProject, sourceFile, cleanup } = createTestProject(`
        enum MyEnum {
          VALUE_A,
          VALUE_B
        }
      `);

      try {
        const enumDeclaration = sourceFile.getEnum('MyEnum');
        const member = enumDeclaration?.getMember('VALUE_A');
        const symbol = member?.getSymbol();
        expect(symbol).toBeDefined();

        if (symbol) {
          const name = assignSymbolName(symbol);
          expect(name).toBe('MyEnum.VALUE_A');
        }
      } finally {
        cleanup();
      }
    });

    it('should handle default export functions', () => {
      const { project: testProject, sourceFile, cleanup } = createTestProject(`
        export default function myDefaultFunction() {
          return 42;
        }
      `);

      try {
        const functionDeclaration = sourceFile.getFunction('myDefaultFunction');
        const symbol = functionDeclaration?.getSymbol();
        expect(symbol).toBeDefined();

        if (symbol) {
          const name = assignSymbolName(symbol);
          expect(name).toBe('myDefaultFunction');
        }
      } finally {
        cleanup();
      }
    });
  });
});