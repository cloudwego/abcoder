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
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
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

  analyze(options: { loadExternalSymbols?: boolean, noDist?: boolean, srcPatterns?: string[] } = {}): TypeScriptStructure {
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


  private analyzePackages(module: ModuleInfo, options: { noDist?: boolean, srcPatterns?: string[] } = {}): PackageInfo[] {
    const sourceFiles = this.findSourceFiles(module, options);
    return sourceFiles.map(file => this.createPackageFromFile(module, file));
  }

  private findSourceFiles(module: ModuleInfo, options: { noDist?: boolean, srcPatterns?: string[] } = {}): string[] {
    // Handle default srcPatterns if not provided
    if (!options.srcPatterns || options.srcPatterns.length === 0) {
      options.srcPatterns = ['**/*.ts', '**/*.js'];
    }

    const allFiles = new Set<string>();

    // 1. Handle srcPatterns if provided
    if (options.srcPatterns && options.srcPatterns.length > 0) {
      // For now, if patterns are provided, we search the entire module for matching files
      // In a more complex implementation, we might want to support actual glob matching
      // Here we reuse findTypeScriptFiles which already finds .ts/.js files
      const files = this.findTypeScriptFiles(module.path, options);
      files.forEach(f => allFiles.add(f));
      return Array.from(allFiles);
    }

    // 2. Original behavior fallback
    // Get tsconfig.json configuration
    const config = this.tsConfigCache.getTsConfig(module.path);

    // Default: all files in tsconfig.json
    if(config.fileNames && config.fileNames.length > 0) {
      config.fileNames.forEach(file => {
        if (fs.existsSync(file)) {
          allFiles.add(file);
        }
      });
      if (allFiles.size > 0) {
        return Array.from(allFiles);
      }
    }
    
    // Fallback to rootDir and outDir
    const searchDirs: string[] = [];
    if (config.rootDir) {
      searchDirs.push(path.join(module.path, config.rootDir));
    }
    if (config.outDir && !(options.noDist && config.outDir === 'dist')) {
      searchDirs.push(path.join(module.path, config.outDir));
    }

    // Default source directories
    const defaultDirs = ['src', 'lib'];
    if (!options.noDist) {
      defaultDirs.push('dist');
    }
    
    for (const dir of defaultDirs) {
      const dirPath = path.join(module.path, dir);
      if (fs.existsSync(dirPath) && fs.statSync(dirPath).isDirectory()) {
        searchDirs.push(dirPath);
      }
    }

    // Find all files in the collected directories
    for (const dir of [...new Set(searchDirs)]) {
      if (fs.existsSync(dir)) {
        const files = this.findTypeScriptFiles(dir, options);
        files.forEach(f => allFiles.add(f));
      }
    }

    return Array.from(allFiles);
  }

  private createPackageFromFile(module: ModuleInfo, file: string): PackageInfo {
    // Calculate file path relative to the package root (module path)
    const relativeFilePath = path.relative(module.path, file);
    const pkgPath = relativeFilePath.replace(/\\/g, '/');

    // Check if this is a main file (index or main)
    const baseName = path.basename(file, path.extname(file));
    const isMain = baseName === 'index' || baseName === 'main';

    // Check if this is a test file
    const isTest = relativeFilePath.includes('test') ||
                   relativeFilePath.includes('__tests__') ||
                   file.includes('.test.') ||
                   file.includes('.spec.');

    return {
      pkgPath,
      moduleName: module.name,
      isMain,
      isTest,
      files: [file], // Each package contains only one file
      imports: [] // Will be populated during parsing
    };
  }

  private findTypeScriptFiles(dir: string, options: { noDist?: boolean, srcPatterns?: string[] } = {}): string[] {
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