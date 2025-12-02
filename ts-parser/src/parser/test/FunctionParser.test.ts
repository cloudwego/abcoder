import { describe, it, expect } from '@jest/globals';
import path from 'path';
import { FunctionParser } from '../FunctionParser';
import { createTestProject, createTestProjectWithMultipleFiles, expectToBeDefined } from './test-utils';

describe('FunctionParser', () => {
  describe('parseFunctions', () => {
    it('should parse function declarations', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        function simpleFunction() {
          return 'hello';
        }
        
        export function exportedFunction(param: string): number {
          return param.length;
        }
        
        function defaultFunction() {
          return 'default';
        }

        export default defaultFunction
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      
      expect(functions['simpleFunction']).toBeDefined();
      expect(functions['exportedFunction']).toBeDefined();
      expect(functions['defaultFunction']).toBeDefined();
      
      expect(functions['simpleFunction']?.Exported).toBe(false);
      expect(functions['exportedFunction']?.Exported).toBe(true);
      expect(functions['defaultFunction']?.Exported).toBe(true);
      
      cleanup();
    });

    it('should parse function declarations 2', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        function simpleFunction() {
          return 'hello';
        }
        
        export function exportedFunction(param: string): number {
          return param.length;
        }
        
        export default function defaultFunction() {
          return 'default';
        }
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      
      expect(functions['simpleFunction']).toBeDefined();
      expect(functions['exportedFunction']).toBeDefined();
      expect(functions['defaultFunction']).toBeDefined();
      
      expect(functions['simpleFunction']?.Exported).toBe(false);
      expect(functions['exportedFunction']?.Exported).toBe(true);
      expect(functions['defaultFunction']?.Exported).toBe(true);
      
      cleanup();
    });

    it('should parse class methods', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        class TestClass {
          public method1() {
            return 'method1';
          }
          
          private method2(): number {
            return 42;
          }
          
          static staticMethod() {
            return 'static';
          }
          
          constructor(private value: string) {
            this.value = value;
          }
        }
        
        export class ExportedClass {
          exportedMethod() {
            return 'exported';
          }
        }
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      
      expect(functions['TestClass.method1']).toBeDefined();
      expect(functions['TestClass.method2']).toBeDefined();
      expect(functions['TestClass.staticMethod']).toBeDefined();
      expect(functions['TestClass.__constructor']).toBeDefined();
      expect(functions['ExportedClass.exportedMethod']).toBeDefined();
      
      expect(functions['TestClass.method1']?.IsMethod).toBe(true);
      expect(functions['TestClass.method1']?.Receiver?.Type.Name).toBe('TestClass');
      
      cleanup();
    });

    it('should parse arrow functions assigned to variables', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        const arrowFunc = (x: number, y: number) => x + y;
        
        const complexArrow = (param: string) => {
          return param.toUpperCase();
        };
        
        export const exportedArrow = () => 'exported';
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      
      expect(functions['arrowFunc']).toBeDefined();
      expect(functions['complexArrow']).toBeDefined();
      expect(functions['exportedArrow']).toBeDefined();
      
      expect(functions['arrowFunc']?.IsMethod).toBe(false);
      expect(functions['exportedArrow']?.Exported).toBe(true);
      
      cleanup();
    });

    it('should parse interface methods', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        interface TestInterface {
          method1(): string;
          method2(param: number): boolean;
        }
        
        export interface ExportedInterface {
          exportedMethod(): void;
        }
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);
      
      expect(functions['TestInterface.method1']).toBeDefined();
      expect(functions['TestInterface.method2']).toBeDefined();
      expect(functions['ExportedInterface.exportedMethod']).toBeDefined();
      
      expect(functions['TestInterface.method1']?.IsInterfaceMethod).toBe(true);
      expect(functions['TestInterface.method1']?.Exported).toBe(true);
      
      cleanup();
    });

    it('should extract function signatures', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        function withParams(a: string, b: number): boolean {
          return a.length > b;
        }
        
        const arrowWithTypes = (x: string): string => x.toUpperCase();
        
        class SignatureTest {
          methodWithGenerics<T>(param: T): T {
            return param;
          }
        }
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);
      
      expect(functions['withParams']?.Signature).toContain('function withParams');
      expect(functions['arrowWithTypes']?.Signature).toContain('=>');
      expect(functions['SignatureTest.methodWithGenerics']?.Signature).toContain('methodWithGenerics');
      
      cleanup();
    });
  });

  describe('extractFunctionCalls', () => {
    it('should extract function calls from function bodies', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        function caller() {
          simpleFunction();
          anotherFunction(42, 'test');
          return nestedCall();
        }
        
        function simpleFunction() {}
        function anotherFunction(a: number, b: string) {}
        function nestedCall() { return 'result'; }
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);
      
      const caller = expectToBeDefined(functions['caller']);
      expect(caller.FunctionCalls).toBeDefined();
      expect(caller.FunctionCalls?.length).toBeGreaterThan(0);
      
      const callNames = expectToBeDefined(caller.FunctionCalls).map(call => call.Name);
      expect(callNames).toContain('simpleFunction');
      expect(callNames).toContain('anotherFunction');
      expect(callNames).toContain('nestedCall');
      
      cleanup();
    });

    it('should handle cross-module function calls', () => {
      const { project, sourceFile, cleanup } = createTestProjectWithMultipleFiles({
        'test.ts': `
          import { externalFunc } from './external';
          
          function usesExternal() {
            externalFunc();
          }
        `,
        'external.ts': `
          export function externalFunc() {
            return 'external';
          }
        `
      });
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);
      
      const usesExternal = expectToBeDefined(functions['usesExternal']);
      expect(usesExternal.FunctionCalls).toBeDefined();
      
      cleanup();
    });
  });

  describe('extractMethodCalls', () => {
    it('should extract method calls from function bodies', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        class Service {
          method1() { return 'service1'; }
          static staticMethod() { return 'static'; }
        }
        
        function user() {
          const service = new Service();
          service.method1();
          Service.staticMethod();
        }
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);
      
      const user = expectToBeDefined(functions['user']);
      expect(user.MethodCalls).toBeDefined();
      
      cleanup();
    });

    it('should handle chained method calls', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        class Chain {
          first() { return this; }
          second() { return this; }
          third() { return 'done'; }
        }

        function chainUser() {
          new Chain().first().second().third();
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)


      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const chainUser = expectToBeDefined(functions['chainUser']);
      expect(chainUser.MethodCalls).toBeDefined();

      cleanup();
    });

    it('should extract method calls on parameter objects', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        interface Context {
          logError(message: string, error: any): void;
          render(template: string, data: any): Promise<void>;
          status: number;
        }

        function getConfig(key: string, ctx: any): Promise<any> {
          return Promise.resolve({ value: 'default' });
        }

        async function handleRequest(ctx: Context, error: any) {
          if (error.code === 404) {
            ctx.status = 404;
            return;
          }
          ctx.logError('Error occurred: ' + error.message, error);
          const config = await getConfig('app.config', ctx);
          await ctx.render('error-page', { message: config.value || error.message || '' });
          ctx.status = 500;
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const handleRequest = expectToBeDefined(functions['handleRequest']);
      expect(handleRequest.MethodCalls).toBeDefined();

      const methodCallNames = expectToBeDefined(handleRequest.MethodCalls).map(call => call.Name);
      // Method calls on parameter objects should include the interface name prefix
      expect(methodCallNames).toContain('Context.logError');
      expect(methodCallNames).toContain('Context.render');

      // Should also have function calls
      expect(handleRequest.FunctionCalls).toBeDefined();
      const functionCallNames = expectToBeDefined(handleRequest.FunctionCalls).map(call => call.Name);
      expect(functionCallNames).toContain('getConfig');

      cleanup();
    });
  });

  describe('extractTypeReferences', () => {
    it('should extract type references from function signatures', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        class CustomType {}
        interface CustomInterface {}
        
        function usesTypes(param: CustomType): CustomInterface {
          return {} as CustomInterface;
        }
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);
      
      const usesTypes = expectToBeDefined(functions['usesTypes']);
      expect(usesTypes.Types).toBeDefined();
      
      const typeNames = expectToBeDefined(usesTypes.Types).map(type => type.Name);
      expect(typeNames).toContain('CustomType');
      expect(typeNames).toContain('CustomInterface');
      
      cleanup();
    });
  });

  describe('extractGlobalVarReferences', () => {
    it('should extract global variable references', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        const globalVar = 'global';
        let mutableGlobal = 42;
        
        function usesGlobals() {
          console.log(globalVar);
          mutableGlobal += 1;
          return globalVar + mutableGlobal;
        }
        
        function localOnly() {
          const localVar = 'local';
          return localVar;
        }
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);
      
      const usesGlobals = expectToBeDefined(functions['usesGlobals']);
      expect(usesGlobals.GlobalVars).toBeDefined();
      
      const globalNames = expectToBeDefined(usesGlobals.GlobalVars).map(gv => gv.Name);
      expect(globalNames).toContain('globalVar');
      expect(globalNames).toContain('mutableGlobal');
      
      const localOnly = expectToBeDefined(functions['localOnly']);
      expect(localOnly.GlobalVars).toHaveLength(0);
      
      cleanup();
    });

    it('should skip built-in globals', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        function usesBuiltins() {
          console.log('test');
          Math.max(1, 2, 3);
          JSON.stringify({});
        }
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);
      
      const usesBuiltins = expectToBeDefined(functions['usesBuiltins']);
      const globalNames = expectToBeDefined(usesBuiltins.GlobalVars).map(gv => gv.Name);
      
      expect(globalNames).not.toContain('console');
      expect(globalNames).not.toContain('Math');
      expect(globalNames).not.toContain('JSON');
      
      cleanup();
    });
  });

  describe('edge cases', () => {
    it('should handle anonymous functions', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        const obj = {
          method: function() {
            return 'anonymous';
          }
        };
        
        const arrow = () => {
          return () => 'nested';
        };
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);
      
      expect(Object.keys(functions)).toHaveLength(1);
      
      cleanup();
    });

    it('should handle destructuring in parameters', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        function destructuredParams({ a, b }: { a: string; b: number }) {
          return a + b;
        }
        
        function arrayParams([x, y]: [string, number]) {
          return x + y;
        }
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);
      
      expect(functions['destructuredParams']).toBeDefined();
      expect(functions['arrayParams']).toBeDefined();
      
      cleanup();
    });

    it('should handle generic functions', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        function genericFunction<T>(param: T): T {
          return param;
        }
        
        class GenericClass<T> {
          method<U>(param: U): T {
            return {} as T;
          }
        }
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);
      
      expect(functions['genericFunction']).toBeDefined();
      expect(functions['GenericClass.method']).toBeDefined();
      
      cleanup();
    });

    it('should give mangled names for 2 symbol located in different files', () => {
      const { project, sourceFile, cleanup } = createTestProjectWithMultipleFiles({
        'file1.ts': `
          export function sharedName() {
            return 'from file1';
          }
        `,
        'file2.ts': `
          export function sharedName() {
            return 'from file2';
          }
        `,
        'test.ts': `
          import { sharedName as sharedName1 } from './file1';
          import { sharedName as sharedName2 } from './file2';

          function testFunction() {
            sharedName1();
            sharedName2();
          }
        `
      });

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const testFunction = expectToBeDefined(functions['testFunction']);
      expect(testFunction.FunctionCalls).toBeDefined();

      const callNames = expectToBeDefined(testFunction.FunctionCalls).map(call => call.Name);
      expect(callNames).toHaveLength(2);
      cleanup();
    });
  });

  describe('type alias dependencies in function parameters and return types', () => {
    it('should extract union type alias dependencies from function parameters', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type Status = 'normal' | 'abnormal';

        export const flipStatus = (s: Status): Status => {
          return s === 'normal' ? 'abnormal' : 'normal';
        };
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      // flipStatus function should exist
      const flipStatus = expectToBeDefined(functions['flipStatus']);
      expect(flipStatus.Exported).toBe(true);

      // Should have Status in Types array
      expect(flipStatus.Types).toBeDefined();
      expect(expectToBeDefined(flipStatus.Types).length).toBeGreaterThan(0);

      const typeNames = expectToBeDefined(flipStatus.Types).map(dep => dep.Name);
      expect(typeNames).toContain('Status');

      cleanup();
    });

    it('should extract type alias dependencies from complex function signatures', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type UserId = string;
        export type UserRole = 'admin' | 'user' | 'guest';

        export type User = {
          id: UserId;
          role: UserRole;
          name: string;
        };

        export function createUser(id: UserId, role: UserRole, name: string): User {
          return { id, role, name };
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const createUser = expectToBeDefined(functions['createUser']);

      // Should have all type aliases in Types array
      expect(createUser.Types).toBeDefined();
      const typeNames = expectToBeDefined(createUser.Types).map(dep => dep.Name);

      expect(typeNames).toContain('UserId');
      expect(typeNames).toContain('UserRole');
      expect(typeNames).toContain('User');

      cleanup();
    });

    it('should not include primitive types in Types array', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export function processData(name: string, age: number, active: boolean): void {
          console.log(name, age, active);
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const processData = expectToBeDefined(functions['processData']);

      // Should not have primitive types
      const typeNames = (processData.Types || []).map(dep => dep.Name);
      expect(typeNames).not.toContain('string');
      expect(typeNames).not.toContain('number');
      expect(typeNames).not.toContain('boolean');
      expect(typeNames).not.toContain('void');

      cleanup();
    });

    it('should extract type aliases from arrow function parameters and return types', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type Result<T> = { success: true; data: T } | { success: false; error: string };
        export type UserData = { name: string; email: string };

        export const fetchUser = async (id: string): Promise<Result<UserData>> => {
          return { success: true, data: { name: 'John', email: 'john@example.com' } };
        };
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const fetchUser = expectToBeDefined(functions['fetchUser']);

      // Should have type aliases in Types array
      expect(fetchUser.Types).toBeDefined();
      const typeNames = expectToBeDefined(fetchUser.Types).map(dep => dep.Name);

      expect(typeNames).toContain('Result');
      expect(typeNames).toContain('UserData');

      cleanup();
    });

    it('should handle multiple occurrences of the same type alias', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type Status = 'active' | 'inactive';

        export function updateStatus(oldStatus: Status, newStatus: Status): Status {
          return newStatus;
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const updateStatus = expectToBeDefined(functions['updateStatus']);

      // Should have Status only once (deduplication)
      expect(updateStatus.Types).toBeDefined();
      const typeNames = expectToBeDefined(updateStatus.Types).map(dep => dep.Name);
      const statusCount = typeNames.filter(name => name === 'Status').length;

      expect(statusCount).toBe(1);

      cleanup();
    });

    it('should filter out self-referencing recursive types', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type TreeNode = {
          value: string;
          children: TreeNode[];
        };

        export function processTree(node: TreeNode): void {
          console.log(node.value);
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const processTree = expectToBeDefined(functions['processTree']);

      // Should have TreeNode in Types array
      expect(processTree.Types).toBeDefined();
      const typeNames = expectToBeDefined(processTree.Types).map(dep => dep.Name);

      expect(typeNames).toContain('TreeNode');

      // TreeNode should only appear once (the self-reference in TreeNode definition should be filtered)
      const treeNodeCount = typeNames.filter(name => name === 'TreeNode').length;
      expect(treeNodeCount).toBe(1);

      cleanup();
    });

    it('should extract function calls from constructors', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export function TestMiddleware() {
          console.log('Test middleware');
        }

        export default class TestMiddleware2 {
          constructor() {
            TestMiddleware()
          }
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      // Find the constructor function
      const constructorFunc = Object.values(functions).find(f => f.Name.includes('constructor'));
      expect(constructorFunc).toBeDefined();

      const ctor = expectToBeDefined(constructorFunc);

      // Should extract TestMiddleware function call
      expect(ctor.FunctionCalls).toBeDefined();
      const functionCalls = expectToBeDefined(ctor.FunctionCalls);
      expect(functionCalls.length).toBeGreaterThan(0);

      const callNames = functionCalls.map(call => call.Name);
      expect(callNames).toContain('TestMiddleware');

      cleanup();
    });
  });

  describe('parameter and return type dependencies', () => {
    it('should extract parameter type dependencies', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type UserType = {
          id: number;
          name: string;
        };

        export type ResultType = {
          success: boolean;
          data: any;
        };

        export function processUser(user: UserType): ResultType {
          return { success: true, data: user };
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const processUser = expectToBeDefined(functions['processUser']);

      // Should extract UserType from parameters
      expect(processUser.Params).toBeDefined();
      const params = expectToBeDefined(processUser.Params);
      expect(params.length).toBeGreaterThan(0);
      const paramNames = params.map(p => p.Name);
      expect(paramNames).toContain('UserType');

      // Should extract ResultType from return type
      expect(processUser.Results).toBeDefined();
      const results = expectToBeDefined(processUser.Results);
      expect(results.length).toBeGreaterThan(0);
      const resultNames = results.map(r => r.Name);
      expect(resultNames).toContain('ResultType');

      cleanup();
    });

    it('should have Receiver field for constructors', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type UserData = {
          id: string;
          name: string;
        };

        export class UserService {
          private data: UserData;

          constructor(userData: UserData) {
            this.data = userData;
          }
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      // Find the constructor
      const constructorFunc = Object.values(functions).find(f => f.Name.includes('constructor'));
      expect(constructorFunc).toBeDefined();

      const ctor = expectToBeDefined(constructorFunc);

      // Should have Receiver field
      expect(ctor.Receiver).toBeDefined();
      expect(ctor.Receiver?.Type.Name).toBe('UserService');
      expect(ctor.IsMethod).toBe(true);

      // Should extract UserData from parameters
      expect(ctor.Params).toBeDefined();
      const params = expectToBeDefined(ctor.Params);
      const paramNames = params.map(p => p.Name);
      expect(paramNames).toContain('UserData');

      cleanup();
    });

    it('should have Receiver field for interface methods', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type RequestType = {
          url: string;
        };

        export type ResponseType = {
          status: number;
        };

        export interface HttpClient {
          request(req: RequestType): ResponseType;
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const requestMethod = expectToBeDefined(functions['HttpClient.request']);

      // Should have Receiver field
      expect(requestMethod.Receiver).toBeDefined();
      expect(requestMethod.Receiver?.Type.Name).toBe('HttpClient');
      expect(requestMethod.IsInterfaceMethod).toBe(true);

      // Should extract RequestType from parameters
      const params = expectToBeDefined(requestMethod.Params);
      const paramNames = params.map(p => p.Name);
      expect(paramNames).toContain('RequestType');

      // Should extract ResponseType from return type
      const results = expectToBeDefined(requestMethod.Results);
      const resultNames = results.map(r => r.Name);
      expect(resultNames).toContain('ResponseType');

      cleanup();
    });

    it('should extract type dependencies from interface methods', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type RequestType = {
          url: string;
        };

        export type ResponseType = {
          status: number;
        };

        export interface HttpClient {
          request(req: RequestType): ResponseType;
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const requestMethod = Object.values(functions).find(f => f.Name.includes('request'));
      expect(requestMethod).toBeDefined();

      const method = expectToBeDefined(requestMethod);

      // Should extract RequestType from parameters
      const params = expectToBeDefined(method.Params);
      const paramNames = params.map(p => p.Name);
      expect(paramNames).toContain('RequestType');

      // Should extract ResponseType from return type
      const results = expectToBeDefined(method.Results);
      const resultNames = results.map(r => r.Name);
      expect(resultNames).toContain('ResponseType');

      cleanup();
    });

    it('should extract type dependencies from arrow functions', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type InputType = string;
        export type OutputType = number;

        export const convert = (input: InputType): OutputType => {
          return parseInt(input);
        };
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const convert = expectToBeDefined(functions['convert']);

      // Should extract InputType from parameters
      const params = expectToBeDefined(convert.Params);
      const paramNames = params.map(p => p.Name);
      expect(paramNames).toContain('InputType');

      // Should extract OutputType from return type
      const results = expectToBeDefined(convert.Results);
      const resultNames = results.map(r => r.Name);
      expect(resultNames).toContain('OutputType');

      cleanup();
    });
  });

  describe('getter and setter support', () => {
    it('should parse getters with type dependencies', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type UserData = {
          id: string;
          name: string;
        };

        export class UserService {
          private data: UserData;

          constructor(userData: UserData) {
            this.data = userData;
          }

          get userData(): UserData {
            return this.data;
          }
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      // Should parse the getter
      const getter = expectToBeDefined(functions['UserService.userData']);
      expect(getter.IsMethod).toBe(true);
      expect(getter.Receiver).toBeDefined();
      expect(getter.Receiver?.Type.Name).toBe('UserService');

      // Should extract return type
      expect(getter.Results).toBeDefined();
      const results = expectToBeDefined(getter.Results);
      const resultNames = results.map(r => r.Name);
      expect(resultNames).toContain('UserData');

      // Should extract global variable reference
      expect(getter.GlobalVars).toBeDefined();

      cleanup();
    });

    it('should parse setters with type dependencies', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type UserData = {
          id: string;
          name: string;
        };

        export class UserService {
          private data: UserData;

          set userData(value: UserData) {
            this.data = value;
          }
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      // Should parse the setter
      const setter = expectToBeDefined(functions['UserService.userData']);
      expect(setter.IsMethod).toBe(true);
      expect(setter.Receiver).toBeDefined();
      expect(setter.Receiver?.Type.Name).toBe('UserService');

      // Should extract parameter type
      expect(setter.Params).toBeDefined();
      const params = expectToBeDefined(setter.Params);
      const paramNames = params.map(p => p.Name);
      expect(paramNames).toContain('UserData');

      cleanup();
    });

    it('should parse getters with function calls', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export function validateData(data: string): boolean {
          return data.length > 0;
        }

        export class DataService {
          private _data: string = '';

          get isValid(): boolean {
            return validateData(this._data);
          }
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const getter = expectToBeDefined(functions['DataService.isValid']);

      // Should extract function call
      expect(getter.FunctionCalls).toBeDefined();
      const functionCalls = expectToBeDefined(getter.FunctionCalls);
      const callNames = functionCalls.map(c => c.Name);
      expect(callNames).toContain('validateData');

      cleanup();
    });

    it('should parse setters with method calls', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export class Logger {
          log(message: string): void {
            console.log(message);
          }
        }

        export class DataService {
          private logger: Logger;
          private _data: string = '';

          constructor() {
            this.logger = new Logger();
          }

          set data(value: string) {
            this._data = value;
            this.logger.log('Data updated');
          }
        }
      `);

      const parser = new FunctionParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const functions = parser.parseFunctions(sourceFile, 'parser-tests', pkgPath);

      const setter = expectToBeDefined(functions['DataService.data']);

      // Should extract method call
      expect(setter.MethodCalls).toBeDefined();
      const methodCalls = expectToBeDefined(setter.MethodCalls);
      const callNames = methodCalls.map(c => c.Name);
      expect(callNames).toContain('Logger.log');

      cleanup();
    });
  });
});