import { SourceFile, Project } from 'ts-morph';
import { Package } from '../types/uniast';
import { FunctionParser } from './FunctionParser';
import { TypeParser } from './TypeParser';
import { VarParser } from './VarParser';

export class PackageParser {
  private functionParser: FunctionParser;
  private typeParser: TypeParser;
  private varParser: VarParser;

  constructor(project: Project, projectRoot: string) {
    this.functionParser = new FunctionParser(project, projectRoot);
    this.typeParser = new TypeParser(projectRoot);
    this.varParser = new VarParser(project, projectRoot);
  }

  async parsePackage(
    sourceFiles: SourceFile[],
    moduleName: string,
    packagePath: string,
    isMain: boolean,
    isTest: boolean
  ): Promise<Package> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const functions: Record<string, any> = {};
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const types: Record<string, any> = {};
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const vars: Record<string, any> = {};

    for (const sourceFile of sourceFiles) {      
      // Parse functions
      const fileFunctions = this.functionParser.parseFunctions(sourceFile, moduleName, packagePath);
      Object.assign(functions, fileFunctions);

      // Parse types
      const fileTypes = this.typeParser.parseTypes(sourceFile, moduleName, packagePath);
      Object.assign(types, fileTypes);

      // Parse variables
      const fileVars = this.varParser.parseVars(sourceFile, moduleName, packagePath);
      Object.assign(vars, fileVars);
    }

    return {
      IsMain: isMain,
      IsTest: isTest,
      PkgPath: packagePath,
      Functions: functions,
      Types: types,
      Vars: vars
    };
  }
}