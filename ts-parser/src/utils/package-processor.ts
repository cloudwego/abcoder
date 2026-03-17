import { Project, ts } from 'ts-morph';
import * as path from 'path';
import * as fs from 'fs';
import { Repository, Module } from '../types/uniast';
import { ModuleParser } from '../parser/ModuleParser';
import { MonorepoPackage } from './monorepo';
import { GraphBuilder } from './graph-builder';
import { TsConfigCache } from './tsconfig-cache';

export interface PackageProcessingOptions {
  loadExternalSymbols?: boolean;
  noDist?: boolean;
  srcPatterns?: string[];
  monorepoMode?: 'combined' | 'separate';
}

export interface PackageProcessingResult {
  success: boolean;
  module?: Module;
  repository?: Repository;
  outputPath?: string;
  error?: Error | { message?: string; stack?: string; name?: string };
  packageInfo: {
    name: string;
    path: string;
    fileCount: number;
    size: number; // bytes
  };
}

/**
 * Package Processor - Encapsulates the complete processing workflow for a single package
 */
export class PackageProcessor {
  private projectRoot: string;

  constructor(projectRoot: string) {
    this.projectRoot = projectRoot;
  }

  /**
   * Process a single package
   */
  async processPackage(
    pkg: MonorepoPackage,
    options: PackageProcessingOptions = {}
  ): Promise<PackageProcessingResult> {
    try {
      // 1. Analyze package information
      const packageInfo = await this.analyzePackage(pkg);

      // 2. Create project instance
      const project = ProjectFactory.createProjectForPackage(pkg.absolutePath, pkg.name);

      // 3. Parse module
      const module = await this.parseModule(project, pkg, options);

      // 4. Create independent repository (for separate mode)
      const repository = this.createPackageRepository(pkg, module);

      // 5. Build graph relationships
      this.buildPackageGraph(repository);

      // 6. Generate output file (only in separate mode)
      let outputPath: string | undefined;
      if (options.monorepoMode !== 'combined') {
        outputPath = await this.generateOutput(pkg, repository);
      }

      return {
        success: true,
        module,
        repository,
        outputPath,
        packageInfo,
      };
    } catch (error) {
      return {
        success: false,
        error: error as Error,
        packageInfo: {
          name: pkg.name || path.basename(pkg.absolutePath),
          path: pkg.path,
          fileCount: 0,
          size: 0,
        },
      };
    }
  }

  /**
   * Analyze package information
   */
  private async analyzePackage(
    pkg: MonorepoPackage
  ): Promise<PackageProcessingResult['packageInfo']> {
    const packageInfo = {
      name: pkg.name || path.basename(pkg.absolutePath),
      path: pkg.path,
      fileCount: 0,
      size: 0,
    };

    try {
      // Recursively collect file information
      const stats = await this.getDirectoryStats(pkg.absolutePath);
      packageInfo.fileCount = stats.fileCount;
      packageInfo.size = stats.totalSize;
    } catch (error) {
      console.warn(`Failed to analyze package ${packageInfo.name}:`, error);
    }

    return packageInfo;
  }

  /**
   * Get directory statistics
   */
  private async getDirectoryStats(
    dirPath: string
  ): Promise<{ fileCount: number; totalSize: number }> {
    let fileCount = 0;
    let totalSize = 0;

    const processDirectory = async (currentPath: string): Promise<void> => {
      try {
        const entries = await fs.promises.readdir(currentPath, { withFileTypes: true });

        for (const entry of entries) {
          const fullPath = path.join(currentPath, entry.name);

          // Skip commonly ignored directories
          if (entry.isDirectory()) {
            if (!PackageProcessor.shouldIgnoreDirectory(entry.name, this.projectRoot)) {
              await processDirectory(fullPath);
            }
          } else if (entry.isFile()) {
            // Only count relevant file types
            if (PackageProcessor.isRelevantFile(entry.name)) {
              const stats = await fs.promises.stat(fullPath);
              fileCount++;
              totalSize += stats.size;
            }
          }
        }
      } catch (error) {
        // Ignore permission errors etc.
        console.debug(`Cannot access directory ${currentPath}:`, error);
      }
    };

    await processDirectory(dirPath);
    return { fileCount, totalSize };
  }



  /**
   * Parse module
   */
  private async parseModule(
    project: Project,
    pkg: MonorepoPackage,
    options: PackageProcessingOptions
  ): Promise<Module> {
    const moduleParser = new ModuleParser(project, this.projectRoot);
    return await moduleParser.parseModule(pkg.absolutePath, pkg.path, options);
  }

  /**
   * Create package repository
   */
  private createPackageRepository(pkg: MonorepoPackage, module: Module): Repository {
    return RepositoryFactory.createPackageRepository(pkg, module);
  }

  /**
   * Build package graph relationships
   */
  private buildPackageGraph(repository: Repository): void {
    GraphBuilder.buildGraph(repository);
  }



  /**
   * Generate output file
   */
  private async generateOutput(pkg: MonorepoPackage, repository: Repository): Promise<string> {
    const sanitizedPackageName = (pkg.name || path.basename(pkg.absolutePath)).replace(
      /[/\\:*?"<>|@]/g,
      '_'
    );

    // Create output directory
    const outputDir = path.join(process.cwd(), 'output');
    if (!fs.existsSync(outputDir)) {
      await fs.promises.mkdir(outputDir, { recursive: true });
    }

    // Generate output file
    const outputPath = path.join(outputDir, `${sanitizedPackageName}.json`);
    const jsonOutput = JSON.stringify(repository, null, 2);

    await fs.promises.writeFile(outputPath, jsonOutput, 'utf8');

    console.log(`Package ${pkg.name || pkg.path} written to: ${outputPath}`);
    return outputPath;
  }

  /**
   * Check if directory should be ignored
   */
  private static shouldIgnoreDirectory(dirName: string, projectRoot?: string): boolean {
    const ignoreDirs = [
      'node_modules',
      '.git',
      '.svn',
      'dist',
      'build',
      'coverage',
      '.nyc_output',
      'tmp',
      'temp',
      '.cache',
      '.next',
      '.nuxt',
      'out',
      'public',
      'static',
      '__pycache__',
      '.pytest_cache',
    ];

    // Check basic ignore rules
    if (ignoreDirs.includes(dirName) || dirName.startsWith('.')) {
      return true;
    }

    // Check .gitignore file
    if (projectRoot) {
      return this.isIgnoredByGitignore(dirName, projectRoot);
    }

    return false;
  }

  /**
   * Check if directory is ignored by .gitignore
   */
  private static isIgnoredByGitignore(dirName: string, projectRoot: string): boolean {
    try {
      const gitignorePath = path.join(projectRoot, '.gitignore');
      if (!fs.existsSync(gitignorePath)) {
        return false;
      }

      const gitignoreContent = fs.readFileSync(gitignorePath, 'utf8');
      const ignorePatterns = gitignoreContent
        .split('\n')
        .map(line => line.trim())
        .filter(line => line && !line.startsWith('#')); // Filter empty lines and comments

      for (const pattern of ignorePatterns) {
        // Simple pattern matching: supports * wildcards and directory matching
        if (this.matchGitignorePattern(dirName, pattern)) {
          return true;
        }
      }
    } catch (error) {
      // Ignore errors reading .gitignore file
      console.debug(`Cannot read .gitignore file: ${error}`);
    }

    return false;
  }

  /**
   * Match gitignore pattern
   */
  private static matchGitignorePattern(dirName: string, pattern: string): boolean {
    // Remove trailing /
    const cleanPattern = pattern.replace(/\/$/, '');

    // Exact match
    if (cleanPattern === dirName) {
      return true;
    }

    // Wildcard match
    if (cleanPattern.includes('*')) {
      const regexPattern = cleanPattern.replace(/\./g, '\\.').replace(/\*/g, '.*');
      const regex = new RegExp(`^${regexPattern}$`);
      return regex.test(dirName);
    }

    return false;
  }

  /**
   * Check if file is relevant
   */
  private static isRelevantFile(fileName: string): boolean {
    const relevantExtensions = [
      '.ts',
      '.tsx',
      '.js',
      '.jsx',
      '.vue',
      '.svelte',
      '.astro',
      '.json',
      '.yaml',
      '.yml',
      '.md',
      '.mdx',
      '.css',
      '.scss',
      '.sass',
      '.less',
      '.html',
      '.htm',
    ];
    const ext = path.extname(fileName).toLowerCase();
    return relevantExtensions.includes(ext);
  }
}

/**
 * Project Factory - Centralized Project creation logic
 * Handles TypeScript project creation with various configurations
 */
export class ProjectFactory {
  /**
   * Create a project with default compiler options
   * Used when no tsconfig.json is available or as fallback
   */
  static createDefaultProject(): Project {
    return new Project({
      compilerOptions: {
        target: 99, // ESNext
        module: 1, // CommonJS
        allowJs: true,
        checkJs: false,
        skipLibCheck: true,
        skipDefaultLibCheck: true,
        strict: false,
        noImplicitAny: false,
        strictNullChecks: false,
        strictFunctionTypes: false,
        strictBindCallApply: false,
        strictPropertyInitialization: false,
        noImplicitReturns: false,
        noFallthroughCasesInSwitch: false,
        noUncheckedIndexedAccess: false,
        noImplicitOverride: false,
        noPropertyAccessFromIndexSignature: false,
        allowUnusedLabels: false,
        allowUnreachableCode: false,
        exactOptionalPropertyTypes: false,
        noImplicitThis: false,
        alwaysStrict: false,
        noImplicitUseStrict: false,
        forceConsistentCasingInFileNames: true,
      },
    });
  }

  /**
   * Create a project for a single repository
   * Handles tsconfig.json resolution and project references
   */
  static createProjectForSingleRepo(
    projectRoot: string,
    tsConfigPath?: string,
    tsConfigCache?: TsConfigCache
  ): Project {
    let configPath = path.join(projectRoot, 'tsconfig.json');

    if (tsConfigPath) {
      let absoluteTsConfigPath = tsConfigPath;
      if (!path.isAbsolute(absoluteTsConfigPath)) {
        absoluteTsConfigPath = path.join(projectRoot, absoluteTsConfigPath);
      }
      configPath = absoluteTsConfigPath;
      if (tsConfigCache) {
        tsConfigCache.setGlobalConfigPath(absoluteTsConfigPath);
      }
    }

    if (fs.existsSync(configPath)) {
      const project = new Project({
        tsConfigFilePath: configPath,
        compilerOptions: {
          allowJs: true,
          skipLibCheck: true,
          forceConsistentCasingInFileNames: true,
        },
      });

      // Handle project references
      ProjectFactory.processProjectReferences(project, configPath);
      return project;
    } else {
      return ProjectFactory.createDefaultProject();
    }
  }

  /**
   * Create a project for a package (monorepo scenario)
   * Simpler version that only checks for local tsconfig.json
   */
  static createProjectForPackage(
    packagePath: string,
    packageName?: string
  ): Project {
    const tsConfigPath = path.join(packagePath, 'tsconfig.json');

    if (fs.existsSync(tsConfigPath)) {
      console.log(
        `Creating project for package ${packageName || path.basename(packagePath)} with tsconfig ${tsConfigPath}`
      );
      try {
        return new Project({
          tsConfigFilePath: tsConfigPath,
          compilerOptions: {
            allowJs: true,
            skipLibCheck: true,
            forceConsistentCasingInFileNames: true,
          },
        });
      } catch (error) {
        console.warn(
          `Failed to create project with tsconfig for package ${packageName || path.basename(packagePath)}:`,
          error
        );
        return ProjectFactory.createDefaultProject();
      }
    } else {
      console.log(
        `No tsconfig.json found for package ${packageName || path.basename(packagePath)}, using default config`
      );
      return ProjectFactory.createDefaultProject();
    }
  }

  /**
   * Process TypeScript project references recursively
   * Handles composite projects and project references
   */
  private static processProjectReferences(project: Project, configPath: string): void {
    const tsConfigQueue: string[] = [configPath];
    const processedTsConfigs = new Set<string>();

    while (tsConfigQueue.length > 0) {
      const currentTsConfig = path.resolve(tsConfigQueue.shift()!);
      if (processedTsConfigs.has(currentTsConfig)) {
        continue;
      }
      processedTsConfigs.add(currentTsConfig);

      const tsConfig_ = ts.readConfigFile(currentTsConfig, ts.sys.readFile);
      if (tsConfig_.error) {
        console.warn('parse tsconfig error', tsConfig_.error);
        continue;
      }

      const parsedConfig = ts.parseJsonConfigFileContent(
        tsConfig_.config,
        ts.sys,
        path.dirname(currentTsConfig)
      );

      if (parsedConfig.errors.length > 0) {
        parsedConfig.errors.forEach(err => {
          console.warn('parse tsconfig warning:', err.messageText);
        });
      }

      // Filter out non-existent files and ensure their directories exist
      const existingFiles = parsedConfig.fileNames.filter(fileName => {
        try {
          // Check if file exists
          if (!fs.existsSync(fileName)) {
            return false;
          }
          // Check if parent directory exists
          const parentDir = path.dirname(fileName);
          return fs.existsSync(parentDir);
        } catch (error) {
          // If any error occurs during checking, exclude the file
          return false;
        }
      });

      if (existingFiles.length > 0) {
        try {
          project.addSourceFilesAtPaths(existingFiles);
        } catch (error) {
          console.warn('Failed to add source files:', error);
        }
      }

      const references = parsedConfig.projectReferences;
      if (!references) {
        continue;
      }

      for (const ref of references) {
        const resolvedRef = ts.resolveProjectReferencePath(ref);
        if (resolvedRef.length > 0) {
          const refPath = path.resolve(path.dirname(currentTsConfig), resolvedRef);
          if (fs.existsSync(refPath)) {
            tsConfigQueue.push(refPath);
          }
        }
      }
    }
  }
}

/**
 * Repository Factory - Centralized creation of Repository objects
 */
export class RepositoryFactory {
  private static AST_VERSION = process.env.ABCODER_AST_VERSION || 'unknown';
  private static TOOL_VERSION = process.env.ABCODER_TOOL_VERSION || 'unknown';

  /**
   * Create a repository for a single project/repository
   */
  static createRepository(repoPath: string): Repository {
    const absolutePath = path.resolve(repoPath);
    return {
      ASTVersion: RepositoryFactory.AST_VERSION,
      ToolVersion: RepositoryFactory.TOOL_VERSION,
      id: path.basename(absolutePath),
      Path: absolutePath,
      Modules: {},
      Graph: {},
    };
  }

  /**
   * Create a repository for a monorepo package
   */
  static createPackageRepository(pkg: MonorepoPackage, module: Module): Repository {
    return {
      ASTVersion: RepositoryFactory.AST_VERSION,
      ToolVersion: RepositoryFactory.TOOL_VERSION,
      Path: pkg.absolutePath,
      id: pkg.name || path.basename(pkg.absolutePath),
      Modules: { [module.Name]: module },
      Graph: {},
    };
  }

  /**
   * Create an empty repository with custom id
   */
  static createEmptyRepository(id: string): Repository {
    return {
      ASTVersion: RepositoryFactory.AST_VERSION,
      ToolVersion: RepositoryFactory.TOOL_VERSION,
      Path: '',
      id,
      Modules: {},
      Graph: {},
    };
  }
}
