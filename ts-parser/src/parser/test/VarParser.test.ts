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
      expect(objDeps.Dependencies!.length).toBe(3);

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
});