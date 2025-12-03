import { describe, it, expect } from '@jest/globals';
import path from 'path';
import { TypeParser } from '../TypeParser';
import { createTestProject, expectToBeDefined } from './test-utils';

describe('TypeParser', () => {
  describe('parseTypes', () => {
    it('should parse class declarations', () => {
      const { sourceFile, cleanup } = createTestProject(`
        class SimpleClass {
          prop: string;
          
          method(): void {}
        }
        
        export class ExportedClass {
          public publicProp: number;
          private privateProp: boolean;
          
          public publicMethod(): string { return ''; }
          private privateMethod(): void {}
        }
        
        abstract class AbstractClass {
          abstract abstractMethod(): void;
        }
      `);
      
      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);
      
      expect(types['SimpleClass']).toBeDefined();
      expect(types['ExportedClass']).toBeDefined();
      expect(types['AbstractClass']).toBeDefined();
      
      expect(types['SimpleClass'].TypeKind).toBe('struct');
      expect(types['ExportedClass'].Exported).toBe(true);
      expect(types['SimpleClass'].Methods).toBeDefined();
      expect(expectToBeDefined(types['SimpleClass'].Methods)['SimpleClass.method']).toBeDefined();
      expect(expectToBeDefined(types['ExportedClass'].Methods)['ExportedClass.publicMethod']).toBeDefined();
      
      cleanup();
    });

    it('should parse interface declarations', () => {
      const { sourceFile, cleanup } = createTestProject(`
        interface SimpleInterface {
          prop: string;
          method(): void;
        }
        
        export interface ExportedInterface {
          requiredProp: number;
          optionalProp?: boolean;
          methodWithParams(a: string, b: number): string;
        }
        
        interface GenericInterface<T> {
          value: T;
          getValue(): T;
        }
      `);
      
      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);
      
      expect(types['SimpleInterface']).toBeDefined();
      expect(types['ExportedInterface']).toBeDefined();
      expect(types['GenericInterface']).toBeDefined();
      
      expect(types['SimpleInterface'].TypeKind).toBe('interface');
      expect(types['ExportedInterface'].Exported).toBe(true);
      expect(types['SimpleInterface'].Methods).toBeDefined();
      expect(expectToBeDefined(types['SimpleInterface'].Methods)['SimpleInterface.method']).toBeDefined();
      expect(types['ExportedInterface'].Methods).toBeDefined();
      expect(expectToBeDefined(types['ExportedInterface'].Methods)['ExportedInterface.methodWithParams']).toBeDefined();
      
      cleanup();
    });

    it('should parse type alias declarations', () => {
      const { sourceFile, cleanup } = createTestProject(`
        type StringAlias = string;
        type ObjectAlias = { prop: string; method(): void };
        type UnionAlias = string | number;
        type GenericAlias<T> = Array<T>;
        type ComplexAlias = {
          nested: {
            deep: string;
          };
          array: Array<{ item: number }>;
        };
        
        export type ExportedAlias = string;
      `);
      
      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);
      
      expect(types['StringAlias']).toBeDefined();
      expect(types['ObjectAlias']).toBeDefined();
      expect(types['UnionAlias']).toBeDefined();
      expect(types['GenericAlias']).toBeDefined();
      expect(types['ComplexAlias']).toBeDefined();
      expect(types['ExportedAlias']).toBeDefined();
      
      expect(types['StringAlias'].TypeKind).toBe('typedef');
      expect(types['ExportedAlias'].Exported).toBe(true);
      
      cleanup();
    });

    it('should parse enum declarations', () => {
      const { sourceFile, cleanup } = createTestProject(`
        enum Color {
          Red = 'red',
          Green = 'green',
          Blue = 'blue'
        }
        
        export enum Status {
          Active = 1,
          Inactive = 0,
          Pending
        }
        
        const enum ConstEnum {
          A = 1,
          B = 2
        }
      `);
      
      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);
      
      expect(types['Color']).toBeDefined();
      expect(types['Status']).toBeDefined();
      expect(types['ConstEnum']).toBeDefined();
      
      expect(types['Color'].TypeKind).toBe('enum');
      expect(types['Status'].Exported).toBe(true);
      
      cleanup();
    });
  });

  describe('inheritance and implementation', () => {
    it('should parse class inheritance', () => {
      const { sourceFile, cleanup } = createTestProject(`
        class BaseClass {
          baseProp: string;
        }
        
        class DerivedClass extends BaseClass {
          derivedProp: number;
        }
        
        class MultiLevel extends DerivedClass {
          multiProp: boolean;
        }
      `);
      
      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      
      const derived = expectToBeDefined(types['DerivedClass']);
      const multi = expectToBeDefined(types['MultiLevel']);
      
      expect(expectToBeDefined(derived.Implements).length).toBeGreaterThan(0);
      expect(expectToBeDefined(multi.Implements).length).toBeGreaterThan(0);
      
      cleanup();
    });

    it('should parse interface inheritance', () => {
      const { sourceFile, cleanup } = createTestProject(`
        interface BaseInterface {
          baseProp: string;
          baseMethod(): void;
        }
        
        interface ExtendedInterface extends BaseInterface {
          extendedProp: number;
          extendedMethod(): boolean;
        }
        
        interface MultiLevel extends ExtendedInterface {
          multiProp: boolean;
        }
      `);
      
      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);
      
      const extended = expectToBeDefined(types['ExtendedInterface']);
      const multi = expectToBeDefined(types['MultiLevel']);
      
      expect(expectToBeDefined(extended.Implements).length).toBeGreaterThan(0);
      expect(expectToBeDefined(multi.Implements).length).toBeGreaterThan(0);
      
      cleanup();
    });

    it('should parse class implementing interfaces', () => {
      const { sourceFile, cleanup } = createTestProject(`
        interface FirstInterface {
          firstProp: string;
          firstMethod(): void;
        }
        
        interface SecondInterface {
          secondProp: number;
          secondMethod(): boolean;
        }
        
        class ImplementingClass implements FirstInterface, SecondInterface {
          firstProp: string;
          secondProp: number;
          
          firstMethod(): void {}
          secondMethod(): boolean { return true; }
        }
      `);
      
      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);
      
      const implementing = expectToBeDefined(types['ImplementingClass']);
      expect(expectToBeDefined(implementing.Implements).length).toBeGreaterThan(0);
      
      cleanup();
    });

    it('should parse complex inheritance scenarios', () => {
      const { sourceFile, cleanup } = createTestProject(`
        class BaseClass {
          baseMethod(): void {}
        }
        
        interface BaseInterface {
          baseInterfaceMethod(): void;
        }
        
        interface ExtendedInterface extends BaseInterface {
          extendedInterfaceMethod(): void;
        }
        
        class ComplexClass extends BaseClass implements ExtendedInterface {
          baseInterfaceMethod(): void {}
          extendedInterfaceMethod(): void {}
        }
      `);
      
      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);
      
      const complex = expectToBeDefined(types['ComplexClass']);
      expect(expectToBeDefined(complex.Implements).length).toBeGreaterThan(0);
      
      cleanup();
    });
  });

  describe('type dependencies', () => {
    it('should extract type dependencies from type aliases', () => {
      const { sourceFile, cleanup } = createTestProject(`
        class CustomType {}
        interface CustomInterface {}

        type SimpleAlias = CustomType;
        type ComplexAlias = {
          prop: CustomType;
          method(): CustomInterface;
        };
        type UnionAlias = CustomType | CustomInterface;
        type GenericAlias<T> = Array<CustomType>;
        type NestedAlias = {
          nested: {
            deep: CustomType;
          };
        };
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      const simpleAlias = expectToBeDefined(types['SimpleAlias']);
      const complexAlias = expectToBeDefined(types['ComplexAlias']);
      const unionAlias = expectToBeDefined(types['UnionAlias']);
      const genericAlias = expectToBeDefined(types['GenericAlias']);
      const nestedAlias = expectToBeDefined(types['NestedAlias']);

      // Type aliases should have dependencies in InlineStruct, not Implements
      expect(expectToBeDefined(simpleAlias.InlineStruct).length).toBeGreaterThan(0);
      expect(expectToBeDefined(complexAlias.InlineStruct).length).toBeGreaterThan(0);
      expect(expectToBeDefined(unionAlias.InlineStruct).length).toBeGreaterThan(0);
      expect(expectToBeDefined(genericAlias.InlineStruct).length).toBeGreaterThan(0);
      expect(expectToBeDefined(nestedAlias.InlineStruct).length).toBeGreaterThan(0);

      const allTypeNames = [
        ...expectToBeDefined(simpleAlias.InlineStruct),
        ...expectToBeDefined(complexAlias.InlineStruct),
        ...expectToBeDefined(unionAlias.InlineStruct),
        ...expectToBeDefined(genericAlias.InlineStruct),
        ...expectToBeDefined(nestedAlias.InlineStruct)
      ].map(dep => expectToBeDefined(dep).Name);

      expect(allTypeNames).toContain('CustomType');
      expect(allTypeNames).toContain('CustomInterface');

      cleanup();
    });

    it('should handle primitive types correctly', () => {
      const { sourceFile, cleanup } = createTestProject(`
        class PrimitiveTypes {
          stringProp: string;
          numberProp: number;
          booleanProp: boolean;
          nullProp: null;
          undefinedProp: undefined;
          anyProp: any;
          unknownProp: unknown;
          voidProp: void;
          neverProp: never;
        }
      `);
      
      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);
      
      const primitiveTypes = expectToBeDefined(types['PrimitiveTypes']);
      const primitiveNames = expectToBeDefined(primitiveTypes.Implements).map(dep => dep.Name);
      
      expect(primitiveNames).not.toContain('string');
      expect(primitiveNames).not.toContain('number');
      expect(primitiveNames).not.toContain('boolean');
      expect(primitiveNames).not.toContain('null');
      expect(primitiveNames).not.toContain('undefined');
      expect(primitiveNames).not.toContain('any');
      expect(primitiveNames).not.toContain('unknown');
      expect(primitiveNames).not.toContain('void');
      expect(primitiveNames).not.toContain('never');
      
      cleanup();
    });
  });

  describe('edge cases', () => {
    it('should handle anonymous classes', () => {
      const { sourceFile, cleanup } = createTestProject(`
        const AnonymousClass = class {
          prop: string;
          method(): void {}
        };
        
        const obj = new class {
          value: number;
          getValue(): number { return this.value; }
        }();
      `);
      
      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);
      
      expect(Object.keys(types).length).toBeGreaterThan(0);
      
      cleanup();
    });

    it('should handle generic types', () => {
      const { sourceFile, cleanup } = createTestProject(`
        class GenericClass<T, U> {
          first: T;
          second: U;
          
          getFirst(): T { return this.first; }
          getSecond(): U { return this.second; }
        }
        
        interface GenericInterface<T> {
          value: T;
          getValue(): T;
        }
        
        type GenericType<T> = Array<T>;
        
        class BoundedGeneric<T extends string> {
          value: T;
        }
      `);
      
      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);
      
      expect(types['GenericClass']).toBeDefined();
      expect(types['GenericInterface']).toBeDefined();
      expect(types['GenericType']).toBeDefined();
      expect(types['BoundedGeneric']).toBeDefined();
      
      cleanup();
    });


    it('should handle nested types', () => {
      const { sourceFile, cleanup } = createTestProject(`
        class Level1 {
          level2: {
            level3: {
              value: string;
            };
          };
        }
        
        type NestedType = {
          nested: {
            deep: {
              deeper: string;
            };
          };
        };
        
        interface NestedInterface {
          nested: {
            method(): {
              result: number;
            };
          };
        }
      `);
      
      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);
      
      expect(types['Level1']).toBeDefined();
      expect(types['NestedType']).toBeDefined();
      expect(types['NestedInterface']).toBeDefined();
      
      cleanup();
    });

    it('should handle function types', () => {
      const { sourceFile, cleanup } = createTestProject(`
        type FunctionType = (param: string) => number;
        type MethodType = {
          method(param: string): number;
        };
        type CallbackType = (error: Error | null, result?: string) => void;
        
        class UsesFunctionTypes {
          func: FunctionType;
          method: MethodType;
          callback: CallbackType;
        }
      `);
      
      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);
      
      expect(types['FunctionType']).toBeDefined();
      expect(types['MethodType']).toBeDefined();
      expect(types['CallbackType']).toBeDefined();
      expect(types['UsesFunctionTypes']).toBeDefined();
      
      cleanup();
    });
  });

  describe('type alias dependencies in InlineStruct', () => {
    it('should extract union type alias dependencies into InlineStruct', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export type Status = 'normal' | 'abnormal';

        export type ServerStatus = {
          code: number;
          status: Status;
        };
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      // Status type should exist
      expect(types['Status']).toBeDefined();
      expect(types['Status'].TypeKind).toBe('typedef');

      // ServerStatus should exist
      const serverStatus = expectToBeDefined(types['ServerStatus']);
      expect(serverStatus.TypeKind).toBe('typedef');

      // ServerStatus should have Status in InlineStruct, not Implements
      expect(serverStatus.Implements).toEqual([]);
      expect(expectToBeDefined(serverStatus.InlineStruct).length).toBeGreaterThan(0);

      const inlineStructNames = expectToBeDefined(serverStatus.InlineStruct).map(dep => dep.Name);
      expect(inlineStructNames).toContain('Status');

      cleanup();
    });

    it('should extract complex type alias dependencies into InlineStruct', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export type UserId = string;
        export type UserRole = 'admin' | 'user' | 'guest';

        export type User = {
          id: UserId;
          role: UserRole;
          name: string;
        };

        export type UserWithMetadata = User & {
          createdAt: Date;
          updatedAt: Date;
        };
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      // User type should have dependencies in InlineStruct
      const user = expectToBeDefined(types['User']);
      expect(user.Implements).toEqual([]);
      expect(expectToBeDefined(user.InlineStruct).length).toBeGreaterThan(0);

      const userInlineNames = expectToBeDefined(user.InlineStruct).map(dep => dep.Name);
      expect(userInlineNames).toContain('UserId');
      expect(userInlineNames).toContain('UserRole');

      // UserWithMetadata should have User in InlineStruct
      const userWithMetadata = expectToBeDefined(types['UserWithMetadata']);
      expect(userWithMetadata.Implements).toEqual([]);
      expect(expectToBeDefined(userWithMetadata.InlineStruct).length).toBeGreaterThan(0);

      const metadataInlineNames = expectToBeDefined(userWithMetadata.InlineStruct).map(dep => dep.Name);
      expect(metadataInlineNames).toContain('User');

      cleanup();
    });

    it('should not include primitive types in InlineStruct', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export type Config = {
          host: string;
          port: number;
          enabled: boolean;
        };
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      const config = expectToBeDefined(types['Config']);

      // Should not have primitive types in InlineStruct
      const inlineStructNames = (config.InlineStruct || []).map(dep => dep.Name);
      expect(inlineStructNames).not.toContain('string');
      expect(inlineStructNames).not.toContain('number');
      expect(inlineStructNames).not.toContain('boolean');

      cleanup();
    });

    it('should handle nested type references in InlineStruct', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export type Address = {
          street: string;
          city: string;
        };

        export type ContactInfo = {
          email: string;
          address: Address;
        };

        export type Person = {
          name: string;
          contact: ContactInfo;
        };
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      // ContactInfo should reference Address
      const contactInfo = expectToBeDefined(types['ContactInfo']);
      expect(expectToBeDefined(contactInfo.InlineStruct).length).toBeGreaterThan(0);

      const contactInfoInlineNames = expectToBeDefined(contactInfo.InlineStruct).map(dep => dep.Name);
      expect(contactInfoInlineNames).toContain('Address');

      // Person should reference ContactInfo
      const person = expectToBeDefined(types['Person']);
      expect(expectToBeDefined(person.InlineStruct).length).toBeGreaterThan(0);

      const personInlineNames = expectToBeDefined(person.InlineStruct).map(dep => dep.Name);
      expect(personInlineNames).toContain('ContactInfo');

      cleanup();
    });

    it('should filter out self-referencing recursive types in InlineStruct', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export type TreeNode = {
          value: string;
          children: TreeNode[];
        };

        export type LinkedListNode = {
          data: number;
          next: LinkedListNode | null;
        };
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      // TreeNode should not include itself in InlineStruct (self-reference should be filtered)
      const treeNode = expectToBeDefined(types['TreeNode']);
      const treeNodeInlineNames = (treeNode.InlineStruct || []).map(dep => dep.Name);
      expect(treeNodeInlineNames).not.toContain('TreeNode');

      // LinkedListNode should not include itself in InlineStruct
      const linkedListNode = expectToBeDefined(types['LinkedListNode']);
      const linkedListInlineNames = (linkedListNode.InlineStruct || []).map(dep => dep.Name);
      expect(linkedListInlineNames).not.toContain('LinkedListNode');

      cleanup();
    });

    it('should parse class constructors and static methods in Methods field', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export class TestClass {
          private value: number;

          constructor(initialValue: number) {
            this.value = initialValue;
          }

          // Instance method
          getValue(): number {
            return this.value;
          }

          // Static method
          static createDefault(): TestClass {
            return new TestClass(0);
          }

          // Another static method
          static fromString(str: string): TestClass {
            return new TestClass(parseInt(str, 10));
          }
        }
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      const testClass = expectToBeDefined(types['TestClass']);
      expect(testClass.Methods).toBeDefined();

      const methods = expectToBeDefined(testClass.Methods);

      // Should include instance methods
      expect(methods['TestClass.getValue']).toBeDefined();
      expect(methods['TestClass.getValue'].Name).toBe('TestClass.getValue');

      // Should include constructor
      expect(methods['TestClass.__constructor']).toBeDefined();
      expect(methods['TestClass.__constructor'].Name).toBe('TestClass.__constructor');

      // Should include static methods
      expect(methods['TestClass.createDefault']).toBeDefined();
      expect(methods['TestClass.createDefault'].Name).toBe('TestClass.createDefault');

      expect(methods['TestClass.fromString']).toBeDefined();
      expect(methods['TestClass.fromString'].Name).toBe('TestClass.fromString');

      cleanup();
    });

    it('should parse class expression with constructors and static methods', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export const ClassExpr = class MyClassExpr {
          private name: string;

          constructor(name: string) {
            this.name = name;
          }

          getName(): string {
            return this.name;
          }

          static create(name: string): MyClassExpr {
            return new MyClassExpr(name);
          }
        };
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      const classExpr = expectToBeDefined(types['MyClassExpr']);
      expect(classExpr.Methods).toBeDefined();

      const methods = expectToBeDefined(classExpr.Methods);

      // Should include instance methods
      expect(methods['MyClassExpr.getName']).toBeDefined();

      // Should include constructor
      expect(methods['MyClassExpr.__constructor']).toBeDefined();

      // Should include static methods
      expect(methods['MyClassExpr.create']).toBeDefined();

      cleanup();
    });
  });

  describe('property type dependencies', () => {
    it('should extract property type dependencies from classes', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export type UserRole = 'admin' | 'user';

        export type UserSettings = {
          theme: string;
        };

        export class User {
          role: UserRole;
          settings: UserSettings;
          active: boolean;

          constructor() {
            this.active = true;
          }
        }
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      const userClass = expectToBeDefined(types['User']);

      // Should have property type dependencies in SubStruct
      expect(userClass.SubStruct).toBeDefined();
      const subStruct = expectToBeDefined(userClass.SubStruct);
      const subStructNames = subStruct.map(dep => dep.Name);

      expect(subStructNames).toContain('UserRole');
      expect(subStructNames).toContain('UserSettings');

      cleanup();
    });

    it('should extract property type dependencies from interfaces', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export type Address = {
          street: string;
          city: string;
        };

        export type PhoneNumber = string;

        export interface Contact {
          address: Address;
          phone: PhoneNumber;
          email: string;
        }
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      const contactInterface = expectToBeDefined(types['Contact']);

      // Should have property type dependencies in SubStruct
      expect(contactInterface.SubStruct).toBeDefined();
      const subStruct = expectToBeDefined(contactInterface.SubStruct);
      const subStructNames = subStruct.map(dep => dep.Name);

      expect(subStructNames).toContain('Address');
      expect(subStructNames).toContain('PhoneNumber');

      cleanup();
    });

    it('should extract property type dependencies from class expressions', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export type ConfigType = {
          timeout: number;
        };

        export const MyClass = class {
          config: ConfigType;

          constructor() {
            this.config = { timeout: 5000 };
          }
        };
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      // Find the class expression (it may have a generated name)
      const classType = Object.values(types).find(t => t.TypeKind === 'struct');
      expect(classType).toBeDefined();

      const myClass = expectToBeDefined(classType);

      // Should have property type dependencies in SubStruct
      expect(myClass.SubStruct).toBeDefined();
      const subStruct = expectToBeDefined(myClass.SubStruct);
      const subStructNames = subStruct.map(dep => dep.Name);

      expect(subStructNames).toContain('ConfigType');

      cleanup();
    });
  });

  describe('getter and setter support in Methods field', () => {
    it('should parse getters in class Methods field', () => {
      const { sourceFile, cleanup } = createTestProject(`
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

          get userId(): string {
            return this.data.id;
          }
        }
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      const userService = expectToBeDefined(types['UserService']);
      expect(userService.Methods).toBeDefined();

      const methods = expectToBeDefined(userService.Methods);

      // Should include getters
      expect(methods['UserService.userData']).toBeDefined();
      expect(methods['UserService.userData'].Name).toBe('UserService.userData');
      expect(methods['UserService.userId']).toBeDefined();
      expect(methods['UserService.userId'].Name).toBe('UserService.userId');

      cleanup();
    });

    it('should parse setters in class Methods field', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export type UserData = {
          id: string;
          name: string;
        };

        export class UserService {
          private data: UserData;

          set userData(value: UserData) {
            this.data = value;
          }

          set userId(value: string) {
            this.data.id = value;
          }
        }
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      const userService = expectToBeDefined(types['UserService']);
      expect(userService.Methods).toBeDefined();

      const methods = expectToBeDefined(userService.Methods);

      // Should include setters
      expect(methods['UserService.userData']).toBeDefined();
      expect(methods['UserService.userData'].Name).toBe('UserService.userData');
      expect(methods['UserService.userId']).toBeDefined();
      expect(methods['UserService.userId'].Name).toBe('UserService.userId');

      cleanup();
    });

    it('should parse both getter and setter with same name', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export class Counter {
          private _count: number = 0;

          get count(): number {
            return this._count;
          }

          set count(value: number) {
            this._count = value;
          }
        }
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      const counter = expectToBeDefined(types['Counter']);
      expect(counter.Methods).toBeDefined();

      const methods = expectToBeDefined(counter.Methods);

      // Should include the count accessor (getter/setter share the same name)
      expect(methods['Counter.count']).toBeDefined();
      expect(methods['Counter.count'].Name).toBe('Counter.count');

      cleanup();
    });

    it('should parse getters in class expressions', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export type Config = {
          timeout: number;
        };

        export const ConfigService = class {
          private _config: Config;

          constructor() {
            this._config = { timeout: 5000 };
          }

          get config(): Config {
            return this._config;
          }
        };
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      // Find the class expression (it will have a name like '__class' or contain 'AnonymousClass')
      const configServiceClass = Object.values(types).find(t =>
        t.TypeKind === 'struct' && t.Methods && Object.keys(t.Methods).some(k => k.includes('config'))
      );
      expect(configServiceClass).toBeDefined();

      const classType = expectToBeDefined(configServiceClass);
      expect(classType.Methods).toBeDefined();

      const methods = expectToBeDefined(classType.Methods);

      // Should include getter (find by searching for config in method name)
      const configMethod = Object.keys(methods).find(k => k.endsWith('.config'));
      expect(configMethod).toBeDefined();

      cleanup();
    });

    it('should parse setters in class expressions', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export type Config = {
          timeout: number;
        };

        export const ConfigService = class {
          private _config: Config;

          set config(value: Config) {
            this._config = value;
          }
        };
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      // Find the class expression (it will have a name like '__class' or contain 'AnonymousClass')
      const configServiceClass = Object.values(types).find(t =>
        t.TypeKind === 'struct' && t.Methods && Object.keys(t.Methods).some(k => k.includes('config'))
      );
      expect(configServiceClass).toBeDefined();

      const classType = expectToBeDefined(configServiceClass);
      expect(classType.Methods).toBeDefined();

      const methods = expectToBeDefined(classType.Methods);

      // Should include setter (find by searching for config in method name)
      const configMethod = Object.keys(methods).find(k => k.endsWith('.config'));
      expect(configMethod).toBeDefined();

      cleanup();
    });
  });

  describe('class property function initializers', () => {
    it('should parse arrow function properties as methods', () => {
      const { sourceFile, cleanup } = createTestProject(`
        class MyClass {
          // Arrow function property
          arrowMethod = (x: number) => {
            return x * 2;
          }

          // Regular method for comparison
          regularMethod(x: number) {
            return x * 2;
          }

          // Non-function property
          normalProp: string = 'hello';
        }
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      expect(types['MyClass']).toBeDefined();
      const myClass = expectToBeDefined(types['MyClass']);
      expect(myClass.Methods).toBeDefined();

      const methods = expectToBeDefined(myClass.Methods);

      // Arrow function property should be in methods
      expect(methods['MyClass.arrowMethod']).toBeDefined();
      expect(methods['MyClass.arrowMethod'].Name).toBe('MyClass.arrowMethod');

      // Regular method should also be in methods
      expect(methods['MyClass.regularMethod']).toBeDefined();
      expect(methods['MyClass.regularMethod'].Name).toBe('MyClass.regularMethod');

      // Normal property should not cause arrow function to appear in SubStruct
      expect(myClass.SubStruct).toBeDefined();
      const subStruct = expectToBeDefined(myClass.SubStruct);

      // SubStruct should only contain type dependencies from non-function properties
      // (in this case, none since 'string' is a primitive)
      const arrowMethodInSubStruct = subStruct.find(dep => dep.Name === 'arrowMethod');
      expect(arrowMethodInSubStruct).toBeUndefined();

      cleanup();
    });

    it('should parse function expression properties as methods', () => {
      const { sourceFile, cleanup } = createTestProject(`
        class Calculator {
          // Function expression property
          add = function(a: number, b: number) {
            return a + b;
          }

          // Named function expression
          subtract = function sub(a: number, b: number) {
            return a - b;
          }
        }
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      expect(types['Calculator']).toBeDefined();
      const calculator = expectToBeDefined(types['Calculator']);
      expect(calculator.Methods).toBeDefined();

      const methods = expectToBeDefined(calculator.Methods);

      // Function expression properties should be in methods
      expect(methods['Calculator.add']).toBeDefined();
      expect(methods['Calculator.add'].Name).toBe('Calculator.add');
      expect(methods['Calculator.subtract']).toBeDefined();
      expect(methods['Calculator.subtract'].Name).toBe('Calculator.subtract');

      cleanup();
    });

    it('should handle mixed property types correctly', () => {
      const { sourceFile, cleanup } = createTestProject(`
        interface Config {
          timeout: number;
        }

        class Service {
          // Function properties
          handler = () => { console.log('handled'); }
          processor = function(data: string) { return data; }

          // Regular method
          execute(): void {}

          // Non-function properties with type dependencies
          config: Config;
          value: number = 42;
        }
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      expect(types['Service']).toBeDefined();
      const service = expectToBeDefined(types['Service']);
      expect(service.Methods).toBeDefined();

      const methods = expectToBeDefined(service.Methods);

      // All function-like members should be in methods
      expect(methods['Service.handler']).toBeDefined();
      expect(methods['Service.processor']).toBeDefined();
      expect(methods['Service.execute']).toBeDefined();

      // SubStruct should contain type dependencies from non-function properties
      expect(service.SubStruct).toBeDefined();
      const subStruct = expectToBeDefined(service.SubStruct);

      // Should have Config dependency from the config property
      const configDep = subStruct.find(dep => dep.Name === 'Config');
      expect(configDep).toBeDefined();

      // Should not have dependencies from function properties
      const handlerDep = subStruct.find(dep => dep.Name === 'handler');
      expect(handlerDep).toBeUndefined();

      cleanup();
    });

    it('should parse arrow function properties in class expressions', () => {
      const { sourceFile, cleanup } = createTestProject(`
        export const MyService = class {
          // Arrow function in class expression
          process = (input: string) => {
            return input.toUpperCase();
          }

          // Regular method
          transform(data: string): string {
            return data;
          }
        };
      `);

      const parser = new TypeParser(process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const types = parser.parseTypes(sourceFile, 'parser-tests', pkgPath);

      // Find the class expression
      const serviceClass = Object.values(types).find(t =>
        t.TypeKind === 'struct' && t.Methods && Object.keys(t.Methods).some(k => k.includes('process'))
      );
      expect(serviceClass).toBeDefined();

      const classType = expectToBeDefined(serviceClass);
      expect(classType.Methods).toBeDefined();

      const methods = expectToBeDefined(classType.Methods);

      // Arrow function property should be in methods (find by searching for process in method name)
      const processMethod = Object.keys(methods).find(k => k.endsWith('.process'));
      expect(processMethod).toBeDefined();
      // Regular method should also be in methods
      const transformMethod = Object.keys(methods).find(k => k.endsWith('.transform'));
      expect(transformMethod).toBeDefined();

      cleanup();
    });
  });
});