import { describe, it, expect } from '@jest/globals';
import path from 'path';
import { VarParser } from '../VarParser';
import { createTestProject, expectToBeDefined } from './test-utils';

describe('VarParser', () => {
  describe('parseVars', () => {
    it('should parse variable declarations', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        const constVar = 'constant';
        let letVar = 'mutable';
        var varVar = 'old-style';
        
        export const exportedConst = 'exported';
        export default const defaultExport = 'default';
      `);
      
      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);


      
      expect(vars['constVar']).toBeDefined();
      expect(vars['letVar']).toBeDefined();
      expect(vars['varVar']).toBeDefined();
      expect(vars['exportedConst']).toBeDefined();
      expect(vars['defaultExport']).toBeDefined();
      
      expect(vars['constVar'].IsConst).toBe(true);
      expect(vars['letVar'].IsConst).toBe(false);
      expect(vars['exportedConst'].IsExported).toBe(true);
      
      cleanup();
    });

    it('should parse export default at different point', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        const defaultExport = 'default';
        export default defaultExport
      `);
      
      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);

      expect(vars['defaultExport']).toBeDefined();
      expect(vars['defaultExport'].IsExported).toBe(true);
      
      cleanup();
    });

    it('should parse enum members', () => {
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
      
      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);
      
      expect(vars['Color.Red']).toBeDefined();
      expect(vars['Color.Green']).toBeDefined();
      expect(vars['Color.Blue']).toBeDefined();
      expect(vars['Status.Active']).toBeDefined();
      expect(vars['Status.Inactive']).toBeDefined();
      expect(vars['Status.Pending']).toBeDefined();
      expect(vars['ConstEnum.A']).toBeDefined();
      expect(vars['ConstEnum.B']).toBeDefined();
      
      expect(vars['Status.Active'].IsExported).toBe(true);
      expect(vars['Status.Active'].IsConst).toBe(true);
      
      cleanup();
    });
  });

  describe('variable destructuring', () => {
    it('should parse object destructuring', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        const obj = { a: 1, b: 2, c: 3 };
        const { a, b } = obj;
        const { c: renamedC } = obj;
        const { d = 'default' } = obj;
        
        function getObj() {
          return { x: 1, y: 2 };
        }
        const { x, y } = getObj();
      `);
      
      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);
      
      expect(vars['a']).toBeDefined();
      expect(vars['b']).toBeDefined();
      expect(vars['renamedC']).toBeDefined();
      expect(vars['d']).toBeDefined();
      expect(vars['x']).toBeDefined();
      expect(vars['y']).toBeDefined();
      
      cleanup();
    });

    it('should parse array destructuring', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        const arr = [1, 2, 3];
        const [first, second] = arr;
        const [,, third] = arr;
        const [x = 0, y = 1] = [];
        
        function getArr() {
          return ['a', 'b', 'c'];
        }
        const [a, b, c] = getArr();
      `);
      
      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);
      
      expect(vars['first']).toBeDefined();
      expect(vars['second']).toBeDefined();
      expect(vars['third']).toBeDefined();
      expect(vars['x']).toBeDefined();
      expect(vars['y']).toBeDefined();
      expect(vars['a']).toBeDefined();
      expect(vars['b']).toBeDefined();
      expect(vars['c']).toBeDefined();
      
      cleanup();
    });

    it('should parse nested destructuring', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        const complex = {
          user: { name: 'John', age: 30 },
          settings: { theme: 'dark' }
        };
        
        const { user: { name, age }, settings: { theme } } = complex;
        
        const arr = [[1, 2], [3, 4]];
        const [[one, two], [three, four]] = arr;
      `);
      
      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);

      
      expect(vars['name']).toBeDefined();
      expect(vars['age']).toBeDefined();
      expect(vars['theme']).toBeDefined();
      expect(vars['one']).toBeDefined();
      expect(vars['two']).toBeDefined();
      expect(vars['three']).toBeDefined();
      expect(vars['four']).toBeDefined();
      
      cleanup();
    });
  });


  describe('dependency extraction', () => {
    it('should extract dependencies from variable initializers', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        const dep1 = 'dependency1';
        const dep2 = 'dependency2';
        
        const usesDeps = dep1 + dep2;
        const objDeps = { a: dep1, b: dep2 };
        const arrDeps = [dep1, dep2];
        
        function getValue() {
          return 'value';
        }
        const funcDep = getValue();
      `);
      
      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      
      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);
      
      const usesDeps = expectToBeDefined(vars['usesDeps']);
      expect(usesDeps.Dependencies).toBeDefined();
      expect(usesDeps.Dependencies!.length).toBe(2);

      const objDeps = expectToBeDefined(vars['objDeps']);
      expect(objDeps.Dependencies!.length).toBe(2);

      const arrDeps = expectToBeDefined(vars['arrDeps']);
      expect(arrDeps.Dependencies!.length).toBe(2);

      const funcDep = expectToBeDefined(vars['funcDep']);
      expect(funcDep.Dependencies!.length).toBe(1);
      
      cleanup();
    });

    it('should extract dependencies from destructuring', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        const sourceObj = { a: 1, b: 2, c: 3 };
        const sourceArr = [1, 2, 3];
        
        const { a, b } = sourceObj;
        const [x, y] = sourceArr;
      `);
      
      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)
      
      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);
      
      const aVar = expectToBeDefined(vars['a']);
      const bVar = expectToBeDefined(vars['b']);
      const xVar = expectToBeDefined(vars['x']);
      const yVar = expectToBeDefined(vars['y']);
      
      expect(aVar.Dependencies!.length).toBe(1);
      expect(bVar.Dependencies!.length).toBe(1);
      expect(xVar.Dependencies!.length).toBe(1);
      expect(yVar.Dependencies!.length).toBe(1);
      
      cleanup();
    });
  });

  describe('edge cases', () => {
    it('should handle re-exports', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export { someVar } from './other-module';
        export * as namespace from './namespace-module';
      `);

      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile : string = sourceFile.getFilePath()
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/')
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile)

      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);

      expect(vars).toBeDefined();

      cleanup();
    });
  });

  describe('type alias dependencies in variable type annotations', () => {
    it('should extract union type alias dependencies from variable declarations', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type Status = 'normal' | 'abnormal';

        export const currentStatus: Status = 'normal';
        export let mutableStatus: Status = 'abnormal';
      `);

      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);

      // currentStatus should have Status as type dependency
      const currentStatus = expectToBeDefined(vars['currentStatus']);
      expect(currentStatus.Type).toBeDefined();
      expect(currentStatus.Type?.Name).toBe('Status');
      expect(currentStatus.IsExported).toBe(true);

      // mutableStatus should also have Status as type dependency
      const mutableStatus = expectToBeDefined(vars['mutableStatus']);
      expect(mutableStatus.Type).toBeDefined();
      expect(mutableStatus.Type?.Name).toBe('Status');

      cleanup();
    });

    it('should extract complex type alias dependencies from variable declarations', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type UserId = string;
        export type UserRole = 'admin' | 'user' | 'guest';

        export type User = {
          id: UserId;
          role: UserRole;
          name: string;
        };

        export const adminUser: User = {
          id: 'admin-001',
          role: 'admin',
          name: 'Admin'
        };

        export const userId: UserId = 'user-123';
      `);

      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);

      // adminUser should reference User type
      const adminUser = expectToBeDefined(vars['adminUser']);
      expect(adminUser.Type).toBeDefined();
      expect(adminUser.Type?.Name).toBe('User');

      // userId should reference UserId type
      const userId = expectToBeDefined(vars['userId']);
      expect(userId.Type).toBeDefined();
      expect(userId.Type?.Name).toBe('UserId');

      cleanup();
    });

    it('should not include primitive types as type dependencies', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export const name: string = 'John';
        export const age: number = 30;
        export const active: boolean = true;
        export const nothing: null = null;
        export const undef: undefined = undefined;
      `);

      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);

      // None of these should have Type set (primitive types are not tracked)
      expect(vars['name'].Type).toBeUndefined();
      expect(vars['age'].Type).toBeUndefined();
      expect(vars['active'].Type).toBeUndefined();
      expect(vars['nothing'].Type).toBeUndefined();
      expect(vars['undef'].Type).toBeUndefined();

      cleanup();
    });

    it('should extract type aliases from destructured variables', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type Config = {
          host: string;
          port: number;
        };

        export type Status = 'running' | 'stopped';

        export const config: Config = { host: 'localhost', port: 8080 };
        export const status: Status = 'running';
      `);

      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);

      // config should reference Config type
      const config = expectToBeDefined(vars['config']);
      expect(config.Type).toBeDefined();
      expect(config.Type?.Name).toBe('Config');

      // status should reference Status type
      const statusVar = expectToBeDefined(vars['status']);
      expect(statusVar.Type).toBeDefined();
      expect(statusVar.Type?.Name).toBe('Status');

      cleanup();
    });

    it('should handle array and generic type aliases', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export type StringArray = Array<string>;
        export type NumberList = number[];

        export const names: StringArray = ['Alice', 'Bob'];
        export const ages: NumberList = [25, 30];
      `);

      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);

      // names should reference StringArray type
      const names = expectToBeDefined(vars['names']);
      expect(names.Type).toBeDefined();
      expect(names.Type?.Name).toBe('StringArray');

      // ages should reference NumberList type
      const ages = expectToBeDefined(vars['ages']);
      expect(ages.Type).toBeDefined();
      expect(ages.Type?.Name).toBe('NumberList');

      cleanup();
    });

    it('should extract type aliases from interface and class types', () => {
      const { project, sourceFile, cleanup } = createTestProject(`
        export interface UserInterface {
          name: string;
          age: number;
        }

        export class UserClass {
          constructor(public name: string, public age: number) {}
        }

        export const user1: UserInterface = { name: 'Alice', age: 25 };
        export const user2: UserClass = new UserClass('Bob', 30);
      `);

      const parser = new VarParser(project, process.cwd());
      let pkgPathAbsFile: string = sourceFile.getFilePath();
      pkgPathAbsFile = pkgPathAbsFile.split('/').slice(0, -1).join('/');
      const pkgPath = path.relative(process.cwd(), pkgPathAbsFile);

      const vars = parser.parseVars(sourceFile, 'parser-tests', pkgPath);

      // user1 should reference UserInterface
      const user1 = expectToBeDefined(vars['user1']);
      expect(user1.Type).toBeDefined();
      expect(user1.Type?.Name).toBe('UserInterface');

      // user2 should reference UserClass
      const user2 = expectToBeDefined(vars['user2']);
      expect(user2.Type).toBeDefined();
      expect(user2.Type?.Name).toBe('UserClass');

      cleanup();
    });
  });
});