import { PackageProcessor, PackageProcessingOptions, PackageProcessingResult } from './package-processor';
import { MonorepoPackage } from './monorepo';

export interface WorkerMessage {
  packages: MonorepoPackage[];
  options: PackageProcessingOptions;
  projectRoot: string;
}

export interface WorkerResult {
  results: PackageProcessingResult[];
  workerId: number;
}

/**
 * Handle worker process for package processing
 */
export function handleWorkerProcess(): void {
  process.on('message', async (message: WorkerMessage) => {
    const { packages, options, projectRoot } = message;
    
    if (!packages || packages.length === 0) {
      // No more packages, primary process is signaling to exit
      console.log(`Worker ${process.pid} received empty package list, exiting.`);
      process.exit(0);
    }

    console.log(`Worker ${process.pid} received ${packages.length} packages to process`);
    
    const processor = new PackageProcessor(projectRoot);
    const workerResults: PackageProcessingResult[] = [];
    
  for (const pkg of packages) {
    try {
      const result = await processor.processPackage(pkg, options);
      workerResults.push(result);
        
        if (result.success) {
          console.log(`Worker ${process.pid} finished processing package ${pkg.name || pkg.path}`);
        } else {
          console.error(`Worker ${process.pid} failed to process package ${pkg.name || pkg.path}:`, result.error?.message);
        }
      } catch (error) {
        console.error(`Worker ${process.pid} error processing package ${pkg.name || pkg.path}:`, error);
        
        // Ensure error is an instance of Error
        const processedError = error instanceof Error ? error : new Error(String(error));
        
        // Add failed result
        workerResults.push({
          success: false,
          error: processedError,
          packageInfo: {
            name: pkg.name || pkg.path,
            path: pkg.path,
            fileCount: 0,
            size: 0,
          },
        });
    }
  }

  if (process.send) {
    const serializedResults = workerResults.map(r => ({
      ...r,
      error: r.error
        ? {
            message: (r.error as Error).message,
            stack: (r.error as Error).stack,
            name: (r.error as Error).name,
          }
        : undefined,
    }));

    const response: WorkerResult = {
      results: serializedResults as unknown as PackageProcessingResult[],
      workerId: process.pid || 0,
    };
    process.send(response);
  }
    
    console.log(`Worker ${process.pid} finished current batch, awaiting next task or shutdown signal.`);
  });

  process.on('disconnect', () => {
    console.log(`Worker ${process.pid} disconnected, exiting.`);
    process.exit(1);
  });
}