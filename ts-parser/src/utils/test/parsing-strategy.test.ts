import { ParsingStrategySelector, ProjectSizeMetrics, ParseStrategy } from '../parsing-strategy';
import { MonorepoPackage } from '../monorepo';

describe('ParsingStrategySelector', () => {
  describe('evaluateProjectSize', () => {
    it('should evaluate small project correctly', () => {
      const smallPackages: MonorepoPackage[] = [
        {
          name: 'small-package',
          path: 'packages/small',
          absolutePath: '/test/packages/small',
          shouldPublish: true,
        },
      ];

      // Mock the file system calls
      const originalGetTypeScriptFiles = (ParsingStrategySelector as any).getTypeScriptFiles;
      (ParsingStrategySelector as any).getTypeScriptFiles = jest.fn().mockReturnValue([
        'file1.ts', 'file2.ts', 'file3.ts'
      ]);

      const originalEvaluatePackageSize = (ParsingStrategySelector as any).evaluatePackageSize;
      (ParsingStrategySelector as any).evaluatePackageSize = jest.fn().mockReturnValue({
        fileCount: 3,
        sizeBytes: 1024 * 10, // 10KB
      });

      const metrics = ParsingStrategySelector.evaluateProjectSize(smallPackages);

      expect(metrics.totalFiles).toBe(3);
      expect(metrics.totalSizeBytes).toBe(1024 * 10);
      expect(metrics.packageCount).toBe(1);
      expect(metrics.avgFilesPerPackage).toBe(3);
      expect(metrics.hasLargePackages).toBe(false);
      expect(metrics.estimatedMemoryUsageMB).toBe(2); // 3 * 0.5 = 1.5, rounded up to 2

      // Restore original methods
      (ParsingStrategySelector as any).getTypeScriptFiles = originalGetTypeScriptFiles;
      (ParsingStrategySelector as any).evaluatePackageSize = originalEvaluatePackageSize;
    });

    it('should evaluate large project correctly', () => {
      const largePackages: MonorepoPackage[] = [
        {
          name: 'large-package-1',
          path: 'packages/large1',
          absolutePath: '/test/packages/large1',
          shouldPublish: true,
        },
        {
          name: 'large-package-2',
          path: 'packages/large2',
          absolutePath: '/test/packages/large2',
          shouldPublish: true,
        },
      ];

      // Mock large project
      const originalEvaluatePackageSize = (ParsingStrategySelector as any).evaluatePackageSize;
      (ParsingStrategySelector as any).evaluatePackageSize = jest.fn().mockReturnValue({
        fileCount: 600, // Large package with 600 files each
        sizeBytes: 1024 * 1024 * 50, // 50MB each
      });

      const metrics = ParsingStrategySelector.evaluateProjectSize(largePackages);

      expect(metrics.totalFiles).toBe(1200); // 600 * 2
      expect(metrics.totalSizeBytes).toBe(1024 * 1024 * 100); // 100MB total
      expect(metrics.packageCount).toBe(2);
      expect(metrics.avgFilesPerPackage).toBe(600);
      expect(metrics.hasLargePackages).toBe(true); // 600 > 200 threshold
      expect(metrics.largestPackageFiles).toBe(600);
      expect(metrics.estimatedMemoryUsageMB).toBe(600); // 1200 * 0.5

      // Restore original method
      (ParsingStrategySelector as any).evaluatePackageSize = originalEvaluatePackageSize;
    });
  });

  describe('selectParsingStrategy', () => {
    it('should recommend single process for small projects', () => {
      const smallMetrics: ProjectSizeMetrics = {
        totalFiles: 50,
        totalSizeBytes: 1024 * 1024 * 5, // 5MB
        packageCount: 3,
        avgFilesPerPackage: 16.7,
        hasLargePackages: false,
        largestPackageFiles: 20,
        estimatedMemoryUsageMB: 25,
      };

      const strategy = ParsingStrategySelector.selectParsingStrategy(smallMetrics);

      expect(strategy.useCluster).toBe(false);
      expect(strategy.reason).toContain('Using single process mode');
      expect(strategy.recommendedWorkers).toBeUndefined();
      expect(strategy.memoryLimit).toBeUndefined();
    });

    it('should recommend cluster mode for large projects', () => {
      const largeMetrics: ProjectSizeMetrics = {
        totalFiles: 1500, // > 1000 threshold
        totalSizeBytes: 1024 * 1024 * 150, // 150MB > 100MB threshold
        packageCount: 25, // > 20 threshold
        avgFilesPerPackage: 60,
        hasLargePackages: true,
        largestPackageFiles: 300, // > 200 threshold
        estimatedMemoryUsageMB: 750, // > 512MB threshold
      };

      const strategy = ParsingStrategySelector.selectParsingStrategy(largeMetrics);

      expect(strategy.useCluster).toBe(true);
      expect(strategy.reason).toContain('Using cluster mode');
      expect(strategy.recommendedWorkers).toBeGreaterThan(0);
      expect(strategy.memoryLimit).toBeDefined();
    });

    it('should recommend cluster mode for projects with high memory usage', () => {
      const highMemoryMetrics: ProjectSizeMetrics = {
        totalFiles: 800, // < 1000 but high memory
        totalSizeBytes: 1024 * 1024 * 80, // 80MB
        packageCount: 15,
        avgFilesPerPackage: 53,
        hasLargePackages: false,
        largestPackageFiles: 150,
        estimatedMemoryUsageMB: 600, // > 512MB threshold
      };

      const strategy = ParsingStrategySelector.selectParsingStrategy(highMemoryMetrics);

      expect(strategy.useCluster).toBe(true);
      expect(strategy.reason).toContain('High estimated memory usage');
    });
  });

  describe('analyzeProject', () => {
    it('should provide complete analysis for a project', () => {
      const packages: MonorepoPackage[] = [
        {
          name: 'test-package',
          path: 'packages/test',
          absolutePath: '/test/packages/test',
          shouldPublish: true,
        },
      ];

      // Mock medium-sized project
      const originalEvaluatePackageSize = (ParsingStrategySelector as any).evaluatePackageSize;
      (ParsingStrategySelector as any).evaluatePackageSize = jest.fn().mockReturnValue({
        fileCount: 100,
        sizeBytes: 1024 * 1024 * 20, // 20MB
      });

      const analysis = ParsingStrategySelector.analyzeProject(packages);

      expect(analysis.metrics).toBeDefined();
      expect(analysis.strategy).toBeDefined();
      expect(analysis.summary).toBeDefined();
      expect(analysis.summary).toContain('Project Analysis Results');
      expect(analysis.summary).toContain('Total files: 100');
      expect(analysis.summary).toContain('Project size: 20.0MB');
      expect(analysis.summary).toContain('Package count: 1');

      // Restore original method
      (ParsingStrategySelector as any).evaluatePackageSize = originalEvaluatePackageSize;
    });
  });
});