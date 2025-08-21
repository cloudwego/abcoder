import { FunctionParser } from '../FunctionParser';
import { createTestProject, expectToBeDefined } from './test-utils';

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
        
        export default function defaultFunction() {
          return 'default';
        }
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');

      
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
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');

      
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
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');

      
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
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');
      
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
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');
      
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
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');
      
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
      const { project, sourceFile, cleanup } = createTestProject(`
        import { externalFunc } from './external';
        
        function usesExternal() {
          externalFunc();
        }
      `);
      
      const parser = new FunctionParser(project, process.cwd());
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');
      
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
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');
      
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
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');
      
      const chainUser = expectToBeDefined(functions['chainUser']);
      expect(chainUser.MethodCalls).toBeDefined();
      
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
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');
      
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
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');
      
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
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');
      
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
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');
      
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
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');
      
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
      const functions = parser.parseFunctions(sourceFile, 'test-module', 'test-package');
      
      expect(functions['genericFunction']).toBeDefined();
      expect(functions['GenericClass.method']).toBeDefined();
      
      cleanup();
    });
  });
});