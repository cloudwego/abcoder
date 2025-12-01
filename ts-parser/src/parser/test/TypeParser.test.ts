import { describe, it, expect } from '@jest/globals';
import path from 'path';
import { TypeParser } from '../TypeParser';
import { createTestProject, expectToBeDefined } from './test-utils';

describe('TypeParser', () => {
  describe('parseTypes', () => {
    it('should parse class declarations', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
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
      expect(expectToBeDefined(types['SimpleClass'].Methods)['method']).toBeDefined();
      expect(expectToBeDefined(types['ExportedClass'].Methods)['publicMethod']).toBeDefined();
      
      cleanup();
    });

    it('should parse interface declarations', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
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
      expect(expectToBeDefined(types['SimpleInterface'].Methods)['method']).toBeDefined();
      expect(types['ExportedInterface'].Methods).toBeDefined();
      expect(expectToBeDefined(types['ExportedInterface'].Methods)['methodWithParams']).toBeDefined();
      
      cleanup();
    });

    it('should parse type alias declarations', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
      const { project, sourceFile, cleanup } = createTestProject(`
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
  });
});