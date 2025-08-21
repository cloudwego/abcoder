import * as path from 'path';
import * as fs from 'fs';
import { Project, SourceFile, SyntaxKind } from 'ts-morph';
import { Module, Package, Import } from '../types/uniast';
import { PackageParser } from './PackageParser';
import { TypeScriptStructureAnalyzer } from '../utils/typescript-structure';
import { TsConfigCache } from '../utils/tsconfig-cache';
import { PathUtils } from '../utils/path-utils';
import * as JSON5 from 'json5';


export class ModuleParser {
  private project: Project;
  private packageParser: PackageParser;
  private tsConfigCache: TsConfigCache;
  private projectRoot: string;

  constructor(project: Project, projectRoot: string) {
    this.project = project;
    this.projectRoot = projectRoot;
    this.packageParser = new PackageParser(project, projectRoot);
    this.tsConfigCache = TsConfigCache.getInstance();
  }

  async parseModule(modulePath: string, relativeDir: string, options: { loadExternalSymbols?: boolean, noDist?: boolean, srcPatterns?: string[] } = {}): Promise<Module> {
    const packageJsonPath = path.join(modulePath, 'package.json');
    let packageJson: any = {};
    
    if (fs.existsSync(packageJsonPath)) {
      try {
        packageJson = JSON5.parse(fs.readFileSync(packageJsonPath, 'utf8'));
      } catch (error) {
        console.warn(`Failed to parse package.json at ${packageJsonPath}:`, error);
      }
    }

    const moduleName = packageJson.name || path.basename(modulePath);
    const moduleVersion = packageJson.version || '0.0.0';
    const isExternal = relativeDir === '';

    // Analyze TypeScript structure
    const analyzer = new TypeScriptStructureAnalyzer(modulePath);
    const structure = analyzer.analyze({ 
      loadExternalSymbols: options.loadExternalSymbols,
      noDist: options.noDist,
      srcPatterns: options.srcPatterns
    });

    // Parse packages
    const packages: Record<string, Package> = {};
    
    for (const [pkgPath, pkgInfo] of structure.packages) {
      const packageObj = await this.packageParser.parsePackage(
        pkgInfo.files.map(filePath => this.project.addSourceFileAtPath(filePath)),
        moduleName,
        pkgPath,
        pkgInfo.isMain,
        pkgInfo.isTest
      );
      packages[pkgPath] = packageObj;
    }

    // Build dependencies map
    const dependencies: Record<string, string> = {};
    if (packageJson.dependencies) {
      Object.assign(dependencies, packageJson.dependencies);
    }
    if (packageJson.devDependencies) {
      Object.assign(dependencies, packageJson.devDependencies);
    }
    if (packageJson.peerDependencies) {
      Object.assign(dependencies, packageJson.peerDependencies);
    }

    let pathUitl = new PathUtils(this.projectRoot)

    // Build files map with detailed import analysis
    const files: Record<string, any> = {};
    for (const [filePath, fileInfo] of structure.files) {
      const sourceFile = this.project.addSourceFileAtPath(filePath);
      const imports = this.extractImports(sourceFile, modulePath);
      const relativeFilePath = pathUitl.getRelativePath(filePath);
      
      files[relativeFilePath] = {
        Path: relativeFilePath,
        Package: fileInfo.packagePath,
        Imports: imports
      };
    }

    return {
      Language: '', // TypeScript
      Version: moduleVersion,
      Name: moduleName,
      Dir: isExternal ? '' : relativeDir,
      Packages: packages,
      Dependencies: Object.keys(dependencies).length > 0 ? dependencies : undefined,
      Files: Object.keys(files).length > 0 ? files : undefined
    };
  }

  private extractImports(sourceFile: SourceFile, modulePath: string): Array<{ Path: string }> {
    const imports: Array<{ Path: string }> = [];
    const sourceFilePath = sourceFile.getFilePath();
    
    // Cache path aliases for this module
    const pathAliases = this.tsConfigCache.getPathAliases(modulePath);
    
    // Track unique import paths to avoid duplicates
    const uniquePaths = new Set<string>();
    
    // Import declarations
    const importDeclarations = sourceFile.getImportDeclarations();
    for (const importDecl of importDeclarations) {
      const originalPath = importDecl.getModuleSpecifierValue();
      const resolvedPath = importDecl.getModuleSpecifierSourceFile()?.getFilePath();
      if (resolvedPath) {
        let relativePath = path.relative(this.projectRoot, resolvedPath);
        imports.push({ Path: relativePath });
      } else {
        imports.push({ Path: "external:" + originalPath });
      }
    }
    
    // Export declarations (re-exports)
    const exportDeclarations = sourceFile.getExportDeclarations();
    for (const exportDecl of exportDeclarations) {
      const originalPath = exportDecl.getModuleSpecifierValue();
      if (originalPath) {
        const resolvedPath = exportDecl.getModuleSpecifierSourceFile()?.getFilePath();
        if (resolvedPath) {
          let relativePath = path.relative(this.projectRoot, resolvedPath);
          imports.push({ Path: relativePath });
        } else {
          imports.push({ Path: "external:" + originalPath });
        }
      }
    }
    return imports;
  }

}