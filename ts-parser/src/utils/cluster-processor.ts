import cluster, { Worker } from 'cluster';
import os from 'os';
import { MonorepoPackage } from './monorepo';
import { PackageProcessingOptions, PackageProcessingResult } from './package-processor';
import { WorkerMessage, WorkerResult } from './cluster-worker';

const numCPUs = os.cpus().length;
export const MAX_WORKERS = 8;

export interface ClusterProcessingResult {
  success: boolean;
  results: PackageProcessingResult[];
  totalProcessed: number;
  errors: Error[];
}

/**
 * Process packages using cluster workers 
 */
export function processPackagesWithCluster(
  packages: MonorepoPackage[],
  projectRoot: string,
  options: PackageProcessingOptions = {}
): Promise<ClusterProcessingResult> {
  return new Promise(resolve => {
    console.log(`Primary ${process.pid} is running with cluster-based package processing.`);
    
    const initialPackageCount = packages.length;
    if (initialPackageCount === 0) {
      console.log('No packages to process.');
      resolve({
        success: true,
        results: [],
        totalProcessed: 0,
        errors: [],
      });
      return;
    }

    // Split packages into batches for workers
    const packagesToProcessQueue: MonorepoPackage[][] = [];
    const batchSize = Math.max(1, Math.ceil(packages.length / (numCPUs * 2))); // Create more batches than workers
    
    for (let i = 0; i < packages.length; i += batchSize) {
      packagesToProcessQueue.push(packages.slice(i, i + batchSize));
    }

    const results: PackageProcessingResult[] = [];
    const errors: Error[] = [];
    const activeWorkers = new Map<number, {
      worker: Worker;
      currentBatch: MonorepoPackage[] | null;
    }>();
    
    let processedBatchCount = 0;
    const totalBatches = packagesToProcessQueue.length;

    const effectiveMaxWorkers = Math.min(
      totalBatches,
      numCPUs,
      MAX_WORKERS,
    );

    console.log(
      `Distributing ${initialPackageCount} packages in ${totalBatches} batches among up to ${effectiveMaxWorkers} workers.`
    );

    function assignTaskToWorker(worker: Worker) {
      const workerData = activeWorkers.get(worker.id);
      if (!workerData) {
        console.error(`Worker ${worker.process.pid} (ID: ${worker.id}) not found in activeWorkers map.`);
        return;
      }

      if (packagesToProcessQueue.length > 0) {
        const batch = packagesToProcessQueue.shift()!;
        workerData.currentBatch = batch;
        
        const message: WorkerMessage = {
          packages: batch,
          options,
          projectRoot,
        };
        
        console.log(
          `Assigning batch of ${batch.length} packages to worker ${worker.process.pid} (ID: ${worker.id})`
        );
        worker.send(message);
      } else {
        console.log(`No more batches. Signaling worker ${worker.process.pid} (ID: ${worker.id}) to exit.`);
        workerData.currentBatch = null;
        
        const exitMessage: WorkerMessage = {
          packages: [],
          options,
          projectRoot,
        };
        worker.send(exitMessage);
      }
    }

    function launchInitialWorkers() {
      const numToLaunch = Math.min(
        effectiveMaxWorkers - activeWorkers.size,
        packagesToProcessQueue.length,
      );
      
      console.log(`Attempting to launch ${numToLaunch} new worker(s).`);
      
      for (let i = 0; i < numToLaunch; i++) {
        if (activeWorkers.size >= effectiveMaxWorkers) {
          break;
        }
        
        const worker = cluster.fork();
        activeWorkers.set(worker.id, {
          worker,
          currentBatch: null,
        });
        
        console.log(`Forked worker ${worker.process.pid} (ID: ${worker.id}).`);
        assignTaskToWorker(worker);
      }
    }

    let timeoutId: NodeJS.Timeout;

    const cleanupAndResolve = (finalResult: ClusterProcessingResult) => {
      clearTimeout(timeoutId);
      cluster.removeAllListeners('message');
      cluster.removeAllListeners('exit');
      resolve(finalResult);
    };

    const resetTimeout = () => {
      clearTimeout(timeoutId);
      timeoutId = setTimeout(() => {
        console.error('Timeout: No messages received for 15 minutes. Terminating.');
        
        for (const workerData of activeWorkers.values()) {
          console.log(`Killing worker ${workerData.worker.process.pid} (ID: ${workerData.worker.id}) due to timeout.`);
          workerData.worker.kill();
        }
        
        activeWorkers.clear();
        cleanupAndResolve({
          success: false,
          results,
          totalProcessed: results.length,
          errors: [...errors, new Error('Processing timeout')],
        });
      }, 120 * 60 * 1000); // 120 minutes
    };

    cluster.on('message', (worker: Worker, message: WorkerResult) => {
      resetTimeout();
      
      console.log(`Primary received results from worker ${worker.process.pid} (ID: ${worker.id}): ${message.results.length} packages processed`);
      
      const workerInfo = activeWorkers.get(worker.id);

      if (workerInfo?.currentBatch) {
        processedBatchCount++;
        console.log(`Batch processed by worker ${worker.process.pid}. Total batches processed: ${processedBatchCount}/${totalBatches}`);
        workerInfo.currentBatch = null;
      }

      if (message.results) {
        for (const result of message.results) {
          results.push(result);
          if (!result.success && result.error) {
            const err: unknown = result.error as unknown;
            let normalizedError: Error | null = null;

            if (err instanceof Error) {
              normalizedError = err;
            } else if (typeof err === 'object' && err !== null) {
              const maybe = err as { message?: string; stack?: string; name?: string };
              const pkgName = (result as any).packageInfo?.name || (result as any).packageInfo?.path;
              const msg = (maybe.message && String(maybe.message)) || `Worker error${pkgName ? ` in ${pkgName}` : ''}`;
              normalizedError = new Error(msg);
              if (maybe.stack) {
                (normalizedError as any).stack = maybe.stack;
              }
              if (maybe.name) {
                (normalizedError as any).name = maybe.name;
              }
            } else if (typeof err === 'string') {
              normalizedError = new Error(err);
            }

            if (normalizedError) {
              errors.push(normalizedError);
            } else {
              errors.push(new Error('Unknown worker error'));
            }
          }
        }
      }

      if (activeWorkers.has(worker.id)) {
        assignTaskToWorker(worker);
      } else {
        console.warn(`Worker ${worker.process.pid} (ID: ${worker.id}) sent message but is no longer in activeWorkers.`);
      }
    });

    cluster.on('exit', (worker: Worker, code: number, signal: string) => {
      resetTimeout();
      
      const workerPid = worker.process.pid;
      const workerId = worker.id;
      
      console.log(`Worker ${workerPid} (ID: ${workerId}) exited with code ${code} ${signal ? `(signal ${signal})` : ''}.`);

      const workerInfo = activeWorkers.get(workerId);
      activeWorkers.delete(workerId);

      if (workerInfo?.currentBatch) {
        console.error(`Worker ${workerPid} (ID: ${workerId}) exited unexpectedly while processing batch. Re-queueing.`);
        packagesToProcessQueue.unshift(workerInfo.currentBatch);
        // Add an error to the errors array for this unexpected exit
        errors.push(new Error(`Worker ${workerPid} (ID: ${workerId}) exited unexpectedly while processing batch`));
      }

      // Try to launch new workers if there are batches and capacity
      if (packagesToProcessQueue.length > 0 && activeWorkers.size < effectiveMaxWorkers) {
        console.log('A worker exited. Checking if new workers should be launched for remaining batches.');
        resetTimeout();
        launchInitialWorkers();
      }

      // Check for completion
      if (processedBatchCount === totalBatches && packagesToProcessQueue.length === 0) {
        if (activeWorkers.size === 0) {
          console.log('All packages processed and all workers exited.');
          cleanupAndResolve({
            success: errors.length === 0,
            results,
            totalProcessed: results.length,
            errors,
          });
        } else {
          console.log(`All batches processed. Waiting for ${activeWorkers.size} worker(s) to exit.`);
        }
      } else if (activeWorkers.size === 0 && packagesToProcessQueue.length > 0) {
        console.error(`All workers have exited, but ${packagesToProcessQueue.length} batches remain in queue. Processing incomplete.`);
        cleanupAndResolve({
          success: false,
          results,
          totalProcessed: results.length,
          errors: [...errors, new Error('Processing incomplete - workers exited unexpectedly')],
        });
      }
    });

    resetTimeout();
    launchInitialWorkers();
  });
}