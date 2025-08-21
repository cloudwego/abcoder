import * as path from 'path';
import * as fs from 'fs';
import { TsConfigCache } from './tsconfig-cache';
import * as JSON5 from 'json5';


export interface TypeScriptStructure {
  modules: Map<string, ModuleInfo>;
  packages: Map<string, PackageInfo>;
  files: Map<string, FileInfo>;
}

export interface ModuleInfo {
  name: string;
  version: string;
  path: string;
  packageJson: any;
  isExternal: boolean;
}

export interface PackageInfo {
  pkgPath: string;
  moduleName: string;
  isMain: boolean;
  isTest: boolean;
  files: string[];
  imports: string[];
}

export interface FileInfo {
  filePath: string;
  packagePath: string;
  moduleName: string;
  isTest: boolean;
  imports: string[];
}

export class TypeScriptStructureAnalyzer {
  private repoPath: string;
  private tsConfigCache: TsConfigCache;

  constructor(repoPath: string) {
    this.repoPath = path.resolve(repoPath);
    this.tsConfigCache = TsConfigCache.getInstance();
  }

  analyze(options: { loadExternalSymbols?: boolean, noDist?: boolean } = {}): TypeScriptStructure {
    const structure: TypeScriptStructure = {
      modules: new Map(),
      packages: new Map(),
      files: new Map()
    };

    // Only find main module, never parse external modules
    const mainModule = this.findMainModule();
    if (mainModule) {
      structure.modules.set(mainModule.name, mainModule);
      const packages = this.analyzePackages(mainModule, options);
      packages.forEach(pkg => {
        structure.packages.set(pkg.pkgPath, pkg);
      });
    }

    // Map files to packages
    for (const [packagePath, pkg] of structure.packages) {
      pkg.files.forEach(filePath => {
        const fileInfo: FileInfo = {
          filePath,
          packagePath,
          moduleName: pkg.moduleName,
          isTest: pkg.isTest,
          imports: [] // Will be populated by parser
        };
        structure.files.set(filePath, fileInfo);
      });
    }

    return structure;
  }


  private findMainModule(): ModuleInfo | null {
    const packageJsonPath = path.join(this.repoPath, 'package.json');
    if (!fs.existsSync(packageJsonPath)) {
      return null;
    }

    try {
      const packageJson = JSON5.parse(fs.readFileSync(packageJsonPath, 'utf8'));
      return {
        name: packageJson.name || path.basename(this.repoPath),
        version: packageJson.version || '0.0.0',
        path: this.repoPath,
        packageJson,
        isExternal: false
      };
    } catch (error) {
      console.warn(`Failed to parse package.json: ${error}`);
      return null;
    }
  }


  private analyzePackages(module: ModuleInfo, options: { noDist?: boolean } = {}): PackageInfo[] {
    const packages: PackageInfo[] = [];
    const sourceDirs = this.findSourceDirectories(module, options);

    for (const sourceDir of sourceDirs) {
      const packageInfos = this.createPackagesFromDirectory(module, sourceDir, options);
      packages.push(...packageInfos);
    }

    return packages;
  }

  private findSourceDirectories(module: ModuleInfo, options: { noDist?: boolean } = {}): string[] {
    const dirs: string[] = [];
    const packageJson = module.packageJson;

    // Get tsconfig.json configuration
    const config = this.tsConfigCache.getTsConfig(module.path);
    
    if (config.rootDir) {
      dirs.push(path.join(module.path, config.rootDir));
    }
    if (config.outDir && !(options.noDist && config.outDir === 'dist')) {
      dirs.push(path.join(module.path, config.outDir));
    }

    // Check package.json for main entry
    if (packageJson.main) {
      const mainDir = path.dirname(path.join(module.path, packageJson.main));
      if (fs.existsSync(mainDir)) {
        dirs.push(mainDir);
      }
    }

    // Default source directories
    const defaultDirs = ['src', 'lib'];
    if (!options.noDist) {
      defaultDirs.push('dist');
    }
    
    for (const dir of defaultDirs) {
      const dirPath = path.join(module.path, dir);
      if (fs.existsSync(dirPath) && fs.statSync(dirPath).isDirectory()) {
        dirs.push(dirPath);
      }
    }

    // Remove duplicates and ensure paths exist
    return [...new Set(dirs)].filter(dir => fs.existsSync(dir));
  }

  private createPackagesFromDirectory(module: ModuleInfo, sourceDir: string, options: { noDist?: boolean } = {}): PackageInfo[] {
    const packages: PackageInfo[] = [];
    
    // Find all TypeScript files
    const tsFiles = this.findTypeScriptFiles(sourceDir, options);
    
    // Group files by directory (each directory = one package)
    const dirGroups = new Map<string, string[]>();
    
    for (const file of tsFiles) {
      const dir = path.dirname(file);
      if (!dirGroups.has(dir)) {
        dirGroups.set(dir, []);
      }
      dirGroups.get(dir)!.push(file);
    }

    // Create packages from directory groups
    for (const [dir, files] of dirGroups) {
      const relativeDir = path.relative(this.repoPath, dir);
      const pkgPath = (relativeDir === '' ? '.' : relativeDir).replace(/\\/g, '/');
      
      // TODO: REWRITE isMain Logic
      const isMain = files.some(file => 
        path.basename(file, path.extname(file)) === 'index' ||
        path.basename(file, path.extname(file)) === 'main'
      );

      const relativePath = path.relative(this.repoPath, dir);
      
      const isTest = relativePath.includes('test') || relativePath.includes('__tests__') ||
                     files.some(file => file.includes('.test.') || file.includes('.spec.'));

      packages.push({
        pkgPath,
        moduleName: module.name,
        isMain,
        isTest,
        files,
        imports: [] // Will be populated during parsing
      });
    }

    return packages;
  }

  private findTypeScriptFiles(dir: string, options: { noDist?: boolean } = {}): string[] {
    const files: string[] = [];
    
    function traverse(currentDir: string) {
      if (!fs.existsSync(currentDir)) return;
      
      const entries = fs.readdirSync(currentDir, { withFileTypes: true });
      
      for (const entry of entries) {
        const fullPath = path.join(currentDir, entry.name);
        
        if (entry.isDirectory()) {
          // Skip node_modules and hidden directories
          if (entry.name === 'node_modules' || entry.name.startsWith('.')) {
            continue;
          }
          // Skip dist folders if --no-dist is enabled
          if (options.noDist && entry.name === 'dist') {
            continue;
          }
          traverse(fullPath);
        } else if (entry.isFile()) {
          // Include .ts and .js files
          if (entry.name.endsWith('.ts') || entry.name.endsWith('.js') ||
              entry.name.endsWith('.tsx') || entry.name.endsWith('.jsx')) {
            files.push(fullPath);
          }
        }
      }
    }

    traverse(dir);
    return files;
  }
}