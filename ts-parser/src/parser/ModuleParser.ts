import * as path from 'path';
import * as fs from 'fs';
import { Node, Project, SourceFile } from 'ts-morph';
import { Module, Package } from '../types/uniast';
import { PackageParser } from './PackageParser';
import { TypeScriptStructureAnalyzer } from '../utils/typescript-structure';
import { TsConfigCache } from '../utils/tsconfig-cache';
import { PathUtils } from '../utils/path-utils';
import * as JSON5 from 'json5';

// Define a more detailed interface for clarity
interface ImportBinding {
  /** The name of the import as it's used in the current file (e.g., the alias). */
  name: string;
  /** True if this is the default import. */
  isDefault: boolean;
  /** True if this is a namespace import (`* as name`). */
  isNamespace?: boolean;
  /** The original name from the exporting module, if aliased. */
  originalName?: string;
}

interface ExtractedImport {
  Path: string;
  Bindings: ImportBinding[];
}


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
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
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

    const pathUitl = new PathUtils(this.projectRoot)

    // Build files map with detailed import analysis
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
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

  private extractImports(sourceFile: SourceFile, _modulePath: string): ExtractedImport[] {
    const importsMap = new Map<string, ImportBinding[]>();

    const importDeclarations = sourceFile.getImportDeclarations();

    for (const importDecl of importDeclarations) {
      const originalPath = importDecl.getModuleSpecifierValue();
      const resolvedPath = importDecl.getModuleSpecifierSourceFile()?.getFilePath();

      // Determine the final path string (relative or external)
      const finalPath = resolvedPath
        ? path.relative(this.projectRoot, resolvedPath)
        : "external:" + originalPath;

      // This array will hold all bindings for the current import declaration
      const bindings: ImportBinding[] = [];
      const importClause = importDecl.getImportClause();

      if (importClause) {
        // 1. Check for a default import: `import MyDefault from ...`
        const defaultImport = importClause.getDefaultImport();
        if (defaultImport) {
          bindings.push({
            name: defaultImport.getText(),
            isDefault: true,
          });
        }

        // 2. Check for named bindings: `{ ... }` or `* as ...`
        const namedBindings = importClause.getNamedBindings();
        if (namedBindings) {
          // Handle namespace import: `import * as MyNamespace from ...`
          if (Node.isNamespaceImport(namedBindings)) {
            bindings.push({
              name: namedBindings.getName(),
              isDefault: false,
              isNamespace: true,
            });
          }
          // Handle named imports: `import { Name, Other as Alias } from ...`
          else if (Node.isNamedImports(namedBindings)) {
            for (const specifier of namedBindings.getElements()) {
              const aliasNode = specifier.getAliasNode();
              const originalName = specifier.getNameNode().getText();

              if (aliasNode) {
                // It has an alias: `originalName as aliasNode`
                bindings.push({
                  name: aliasNode.getText(), // The name used in this file is the alias
                  isDefault: false,
                  originalName,
                });
              } else {
                // No alias: `{ originalName }`
                bindings.push({
                  name: originalName, // The name is the same as the original
                  isDefault: false,
                });
              }
            }
          }
        }
      }

      // Merge bindings into the map, ensuring no duplicates
      if (!importsMap.has(finalPath)) {
        importsMap.set(finalPath, bindings);
      } else {
        const existingBindings = importsMap.get(finalPath)!;
        const uniqueBindings = new Map<string, ImportBinding>();

        // Add existing bindings to the map for uniqueness
        existingBindings.forEach((binding) => {
          uniqueBindings.set(binding.name, binding);
        });

        // Add new bindings to the map for uniqueness
        bindings.forEach((binding) => {
          uniqueBindings.set(binding.name, binding);
        });

        // Update the map with the merged bindings
        importsMap.set(finalPath, Array.from(uniqueBindings.values()));
      }
    }

    // Convert the map back to an array of ExtractedImport
    return Array.from(importsMap.entries()).map(([Path, Bindings]) => ({ Path, Bindings }));
  }

}