import { Project } from 'ts-morph';
import * as path from 'path';
import * as fs from 'fs';
import * as cluster from 'cluster';
import { Repository } from '../types/uniast';
import { ModuleParser } from './ModuleParser';
import { TsConfigCache } from '../utils/tsconfig-cache';
import { MonorepoUtils, MonorepoPackage } from '../utils/monorepo';
import { processPackagesWithCluster } from '../utils/cluster-processor';
import { handleWorkerProcess } from '../utils/cluster-worker';
import { GraphBuilder } from '../utils/graph-builder';
import { ProjectFactory, RepositoryFactory } from '../utils/package-processor';
import { ParsingStrategySelector } from '../utils/parsing-strategy';

export class RepositoryParser {
  private project?: Project;
  private moduleParser?: ModuleParser;
  private tsConfigCache: TsConfigCache;
  private projectRoot: string;
  private tsConfigPath?: string;

  constructor(projectRoot: string, tsConfigPath?: string) {
    this.tsConfigCache = TsConfigCache.getInstance();
    this.projectRoot = projectRoot;
    this.tsConfigPath = tsConfigPath;
  }

  async parseRepository(
    repoPath: string,
    options: {
      loadExternalSymbols?: boolean;
      noDist?: boolean;
      srcPatterns?: string[];
      monorepoMode?: 'combined' | 'separate';
    } = {}
  ): Promise<Repository> {
    const absolutePath = path.resolve(repoPath);
    const repository: Repository = RepositoryFactory.createRepository(repoPath);

    const isMonorepo = MonorepoUtils.isMonorepo(absolutePath);

    if (isMonorepo) {
      const packages = MonorepoUtils.getMonorepoPackages(absolutePath);
      const monorepoMode = options.monorepoMode || 'combined';
      if (monorepoMode === 'separate') {
        await this.parseMonorepoSeparateMode(packages, repository, options);
      } else {
        // Using combined output mode - all packages will be merged into one JSON file
        await this.parseMonorepoCombinedMode(packages, repository, options);
      }
    } else {
      await this.parseSingleProjectMode(absolutePath, repository, options);
    }

    this.buildGlobalGraph(repository);
    return repository;
  }

  /**
   * Parse a single project (non-monorepo) repository
   */
  private async parseSingleProjectMode(
    absolutePath: string,
    repository: Repository,
    options: {
      loadExternalSymbols?: boolean;
      noDist?: boolean;
      srcPatterns?: string[];
    }
  ): Promise<void> {
    console.log('Single project detected.');
    this.project = ProjectFactory.createProjectForSingleRepo(
      this.projectRoot,
      this.tsConfigPath,
      this.tsConfigCache
    );
    this.moduleParser = new ModuleParser(this.project, this.projectRoot);
    const module = await this.moduleParser.parseModule(absolutePath, '.', options);
    repository.Modules[module.Name] = module;
  }

  private buildGlobalGraph(repository: Repository): void {
    GraphBuilder.buildGraph(repository);
  }

  /**
   * Parse monorepo packages in separate mode with cluster-based parallel processing
   * Uses cluster workers for optimal performance and resource utilization
   */
  private async parseMonorepoSeparateMode(
    packages: MonorepoPackage[],
    repository: Repository,
    options: {
      loadExternalSymbols?: boolean;
      noDist?: boolean;
      srcPatterns?: string[];
      monorepoMode?: 'combined' | 'separate';
      maxConcurrency?: number;
      enableParallel?: boolean;
      useCluster?: boolean;
    }
  ): Promise<void> {
    console.log(`Processing ${packages.length} packages in separate mode (cluster-based parallel)`);

    try {
      // Always use cluster-based processing for optimal performance
      await this.processPackagesWithClusterMode(packages, repository, options);

      console.log(`All packages processed successfully`);
    } catch (error) {
      console.error('Failed to process packages:', error);
      throw error;
    }

    if (global.gc) {
      global.gc();
    }

    // Build global graph for the repository
    this.buildGlobalGraph(repository);
  }

  /**
   * Process packages using cluster workers for better performance
   */
  private async processPackagesWithClusterMode(
    packages: MonorepoPackage[],
    repository: Repository,
    options: any
  ): Promise<void> {
    if ((cluster as any).isPrimary || (cluster as any).isMaster) {
      const result = await processPackagesWithCluster(packages, this.projectRoot, options);

      if (!result.success) {
        throw new Error(
          `Cluster processing failed: ${result.errors.map(e => e.message).join(', ')}`
        );
      }

      // Merge results into main repository
      for (const packageResult of result.results) {
        if (packageResult.success && packageResult.module) {
          repository.Modules[packageResult.module.Name] = packageResult.module;
        }
      }

      console.log(`Cluster processing completed: ${result.totalProcessed} packages processed`);
    } else {
      handleWorkerProcess();
    }
  }

  /**
   * Parse monorepo packages in combined mode - all packages will be merged into one JSON file
   */
  private async parseMonorepoCombinedMode(
    packages: MonorepoPackage[],
    repository: Repository,
    options: {
      loadExternalSymbols?: boolean;
      noDist?: boolean;
      srcPatterns?: string[];
      monorepoMode?: 'combined' | 'separate';
    }
  ): Promise<void> {
    // Analyze project size and select parsing strategy
    const analysis = ParsingStrategySelector.analyzeProject(packages);
    console.log('Project analysis results:');
    console.log(analysis.summary);

    // Choose parsing mode based on strategy
    if (analysis.strategy.useCluster) {
      console.log('Using cluster mode to avoid memory overflow');
      // Use cluster mode but maintain combined mode to avoid outputting individual files
      await this.parseMonorepoSeparateMode(packages, repository, {
        ...options,
        monorepoMode: 'combined', // Ensure no individual package files are output
        useCluster: true,
        maxConcurrency: analysis.strategy.recommendedWorkers,
      });
      // In cluster mode, also need to build global graph for merged output
      this.buildGlobalGraph(repository);
      return;
    }

    // For smaller projects, use traditional combined mode
    for (const pkg of packages) {
      let project: Project;
      const packageTsConfigPath = path.join(pkg.absolutePath, 'tsconfig.json');
      if (fs.existsSync(packageTsConfigPath)) {
        try {
          project = new Project({
            tsConfigFilePath: packageTsConfigPath,
            compilerOptions: {
              allowJs: true,
              skipLibCheck: true,
              forceConsistentCasingInFileNames: true,
            },
          });
        } catch (error) {
          project = ProjectFactory.createDefaultProject();
          console.warn(`Failed to parse package ${pkg.name || pkg.path}:`, error);
        }
      } else {
        project = ProjectFactory.createDefaultProject();
        console.log(`No tsconfig.json found for package ${pkg.name || pkg.path}, skipping.`);
      }
      const moduleParser = new ModuleParser(project, this.projectRoot);
      const module = await moduleParser.parseModule(pkg.absolutePath, pkg.path, options);
      repository.Modules[module.Name] = module;
    }
  }
}
