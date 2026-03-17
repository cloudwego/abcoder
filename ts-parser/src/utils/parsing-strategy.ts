import * as fs from 'fs';
import * as path from 'path';
import { MonorepoPackage } from './monorepo';

export interface ProjectSizeMetrics {
  totalFiles: number;
  totalSizeBytes: number;
  packageCount: number;
  avgFilesPerPackage: number;
  hasLargePackages: boolean;
  largestPackageFiles: number;
  estimatedMemoryUsageMB: number;
}

export interface ParseStrategy {
  useCluster: boolean;
  reason: string;
  recommendedWorkers?: number;
  memoryLimit?: string;
}

export class ParsingStrategySelector {
  // Project size evaluation thresholds
  private static readonly THRESHOLDS = {
    // File count thresholds
    TOTAL_FILES_LARGE: 1000,        // Projects with more than 1000 files are considered large
    TOTAL_FILES_HUGE: 5000,         // Projects with more than 5000 files are considered huge
    
    // Project size thresholds (MB)
    TOTAL_SIZE_LARGE_MB: 100,       // Total size exceeding 100MB
    TOTAL_SIZE_HUGE_MB: 500,        // Total size exceeding 500MB
    
    // Package count thresholds
    PACKAGE_COUNT_LARGE: 20,        // More than 20 packages
    PACKAGE_COUNT_HUGE: 50,         // More than 50 packages
    
    // Single package file count thresholds
    LARGE_PACKAGE_FILES: 200,       // Single package with more than 200 files
    HUGE_PACKAGE_FILES: 500,        // Single package with more than 500 files
    
    // Average files per package threshold
    AVG_FILES_PER_PACKAGE: 100,     // Average files per package exceeding 100
    
    // Memory usage estimation thresholds (MB)
    MEMORY_USAGE_LARGE_MB: 512,     // Estimated memory usage exceeding 512MB
    MEMORY_USAGE_HUGE_MB: 1024,     // Estimated memory usage exceeding 1GB
  };

  /**
   * Evaluate project size and complexity
   */
  static evaluateProjectSize(packages: MonorepoPackage[]): ProjectSizeMetrics {
    let totalFiles = 0;
    let totalSizeBytes = 0;
    let largestPackageFiles = 0;
    let hasLargePackages = false;

    for (const pkg of packages) {
      const packageMetrics = this.evaluatePackageSize(pkg.absolutePath);
      totalFiles += packageMetrics.fileCount;
      totalSizeBytes += packageMetrics.sizeBytes;
      
      if (packageMetrics.fileCount > largestPackageFiles) {
        largestPackageFiles = packageMetrics.fileCount;
      }
      
      if (packageMetrics.fileCount > this.THRESHOLDS.LARGE_PACKAGE_FILES) {
        hasLargePackages = true;
      }
    }

    const packageCount = packages.length;
    const avgFilesPerPackage = packageCount > 0 ? totalFiles / packageCount : 0;
    
    // Estimate memory usage (based on empirical formula)
    // Each file requires approximately 0.5MB memory for parsing and AST storage
    const estimatedMemoryUsageMB = Math.ceil(totalFiles * 0.5);

    return {
      totalFiles,
      totalSizeBytes,
      packageCount,
      avgFilesPerPackage,
      hasLargePackages,
      largestPackageFiles,
      estimatedMemoryUsageMB,
    };
  }

  /**
   * Evaluate the size of a single package
   */
  private static evaluatePackageSize(packagePath: string): { fileCount: number; sizeBytes: number } {
    let fileCount = 0;
    let sizeBytes = 0;

    try {
      const files = this.getTypeScriptFiles(packagePath);
      fileCount = files.length;
      
      for (const file of files) {
        try {
          const stats = fs.statSync(file);
          sizeBytes += stats.size;
        } catch (error) {
          // Ignore inaccessible files
        }
      }
    } catch (error) {
      console.warn(`Failed to evaluate package size for ${packagePath}:`, error);
    }

    return { fileCount, sizeBytes };
  }

  /**
   * Get TypeScript files in the package
   */
  private static getTypeScriptFiles(packagePath: string): string[] {
    const files: string[] = [];
    const extensions = ['.ts', '.tsx', '.js', '.jsx'];

    const scanDirectory = (dir: string) => {
      try {
        const entries = fs.readdirSync(dir, { withFileTypes: true });
        
        for (const entry of entries) {
          const fullPath = path.join(dir, entry.name);
          
          if (entry.isDirectory()) {
            // Skip common non-source directories
            if (!['node_modules', 'dist', 'build', '.git', 'coverage'].includes(entry.name)) {
              scanDirectory(fullPath);
            }
          } else if (entry.isFile()) {
            const ext = path.extname(entry.name);
            if (extensions.includes(ext)) {
              files.push(fullPath);
            }
          }
        }
      } catch (error) {
        // Ignore inaccessible directories
      }
    };

    scanDirectory(packagePath);
    return files;
  }

  /**
   * Select parsing strategy based on project size
   */
  static selectParsingStrategy(metrics: ProjectSizeMetrics): ParseStrategy {
    const {
      totalFiles,
      totalSizeBytes,
      packageCount,
      avgFilesPerPackage,
      hasLargePackages,
      largestPackageFiles,
      estimatedMemoryUsageMB,
    } = metrics;

    const totalSizeMB = totalSizeBytes / (1024 * 1024);

    // Conditions to determine if cluster mode is needed
    const conditions = {
      tooManyFiles: totalFiles > this.THRESHOLDS.TOTAL_FILES_LARGE,
      tooLarge: totalSizeMB > this.THRESHOLDS.TOTAL_SIZE_LARGE_MB,
      tooManyPackages: packageCount > this.THRESHOLDS.PACKAGE_COUNT_LARGE,
      hasLargePackages,
      highAvgFiles: avgFilesPerPackage > this.THRESHOLDS.AVG_FILES_PER_PACKAGE,
      highMemoryUsage: estimatedMemoryUsageMB > this.THRESHOLDS.MEMORY_USAGE_LARGE_MB,
    };

    // Build reason explanations
    const reasons: string[] = [];
    if (conditions.tooManyFiles) {
      reasons.push(`Too many files (${totalFiles} > ${this.THRESHOLDS.TOTAL_FILES_LARGE})`);
    }
    if (conditions.tooLarge) {
      reasons.push(`Project too large (${totalSizeMB.toFixed(1)}MB > ${this.THRESHOLDS.TOTAL_SIZE_LARGE_MB}MB)`);
    }
    if (conditions.tooManyPackages) {
      reasons.push(`Too many packages (${packageCount} > ${this.THRESHOLDS.PACKAGE_COUNT_LARGE})`);
    }
    if (conditions.hasLargePackages) {
      reasons.push(`Large packages exist (largest package ${largestPackageFiles} files > ${this.THRESHOLDS.LARGE_PACKAGE_FILES})`);
    }
    if (conditions.highAvgFiles) {
      reasons.push(`High average files per package (${avgFilesPerPackage.toFixed(1)} > ${this.THRESHOLDS.AVG_FILES_PER_PACKAGE})`);
    }
    if (conditions.highMemoryUsage) {
      reasons.push(`High estimated memory usage (${estimatedMemoryUsageMB}MB > ${this.THRESHOLDS.MEMORY_USAGE_LARGE_MB}MB)`);
    }

    // Determine whether to use cluster mode
    const shouldUseCluster = Object.values(conditions).some(condition => condition);

    if (shouldUseCluster) {
      // Recommend worker count based on project size
      let recommendedWorkers = 2;
      let memoryLimit = '2048';

      if (totalFiles > this.THRESHOLDS.TOTAL_FILES_HUGE || 
          totalSizeMB > this.THRESHOLDS.TOTAL_SIZE_HUGE_MB ||
          estimatedMemoryUsageMB > this.THRESHOLDS.MEMORY_USAGE_HUGE_MB) {
        recommendedWorkers = Math.min(8, Math.ceil(packageCount / 10));
        memoryLimit = '4096';
      } else {
        recommendedWorkers = Math.min(4, Math.ceil(packageCount / 20));
      }

      return {
        useCluster: true,
        reason: `Using cluster mode: ${reasons.join(', ')}`,
        recommendedWorkers,
        memoryLimit,
      };
    } else {
      return {
        useCluster: false,
        reason: `Using single process mode: moderate project size (${totalFiles} files, ${totalSizeMB.toFixed(1)}MB, ${packageCount} packages)`,
      };
    }
  }

  /**
   * Get complete analysis of project parsing strategy
   */
  static analyzeProject(packages: MonorepoPackage[]): {
    metrics: ProjectSizeMetrics;
    strategy: ParseStrategy;
    summary: string;
  } {
    const metrics = this.evaluateProjectSize(packages);
    const strategy = this.selectParsingStrategy(metrics);

    const summary = `
Project Analysis Results:
- Total files: ${metrics.totalFiles}
- Project size: ${(metrics.totalSizeBytes / (1024 * 1024)).toFixed(1)}MB
- Package count: ${metrics.packageCount}
- Average files per package: ${metrics.avgFilesPerPackage.toFixed(1)}
- Largest package files: ${metrics.largestPackageFiles}
- Estimated memory usage: ${metrics.estimatedMemoryUsageMB}MB
- Parsing strategy: ${strategy.reason}
${strategy.recommendedWorkers ? `- Recommended workers: ${strategy.recommendedWorkers}` : ''}
${strategy.memoryLimit ? `- Suggested memory limit: ${strategy.memoryLimit}MB` : ''}
    `.trim();

    return { metrics, strategy, summary };
  }
}