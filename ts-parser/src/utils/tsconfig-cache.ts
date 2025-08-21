import * as fs from 'fs';
import * as path from 'path';
import * as JSON5 from 'json5';
import { ts } from 'ts-morph';


export interface TsConfigData {
  compilerOptions?: any;
  paths?: Record<string, string[]>;
  rootDir?: string;
  outDir?: string;
  fileNames?: string[];
  [key: string]: any;
}

export class TsConfigCache {
  private static instance: TsConfigCache;
  private cache: Map<string, TsConfigData> = new Map();
  private globalConfigPath: string | null = null;

  private constructor() {}

  public static getInstance(): TsConfigCache {
    if (!TsConfigCache.instance) {
      TsConfigCache.instance = new TsConfigCache();
    }
    return TsConfigCache.instance;
  }

  /**
   * Set the global tsconfig.json path
   * @param configPath Custom tsconfig.json path (can be relative)
   */
  public setGlobalConfigPath(configPath: string): void {
    this.globalConfigPath = path.resolve(configPath);
    // Clear cache, force reload
    this.cache.clear();
  }

  /**
   * Get tsconfig.json data, load only once and cache
   * @param projectRoot Project root directory
   * @param customPath Custom tsconfig.json path (optional)
   * @returns tsconfig.json data
   */
  public getTsConfig(projectRoot: string, customPath?: string): TsConfigData {
    // Get tsconfig.json path
    const configPath = customPath || this.globalConfigPath || path.join(projectRoot, 'tsconfig.json');
    const resolvedPath = path.resolve(configPath);
    const emptyConfig: TsConfigData = {};

    // If cache exists, return directly
    if (this.cache.has(resolvedPath)) {
      return this.cache.get(resolvedPath)!;
    }

    // If file does not exist, return empty object
    if (!fs.existsSync(resolvedPath)) {
      this.cache.set(resolvedPath, emptyConfig);
      return emptyConfig;
    }

    try {
      const tsConfig_ = ts.readConfigFile(
        resolvedPath, ts.sys.readFile
      );
      if (tsConfig_.error) {
        console.warn(`Failed to read tsconfig.json at ${this.globalConfigPath}:`, tsConfig_.error);
        return emptyConfig;
      }
      const parsedConfig = ts.parseJsonConfigFileContent(
        tsConfig_.config,
        ts.sys,
        path.dirname(configPath)
      );
      if(parsedConfig.errors.length > 0) {
        parsedConfig.errors.forEach(err => {
          console.warn("parse tsconfig warning:", err.messageText)
        });
      }

      const tsconfig = JSON5.parse(fs.readFileSync(resolvedPath, 'utf8'));
      const configData: TsConfigData = {
        compilerOptions: tsconfig.compilerOptions || {},
        paths: tsconfig.compilerOptions?.paths || {},
        rootDir: tsconfig.compilerOptions?.rootDir,
        outDir: tsconfig.compilerOptions?.outDir,
        fileNames: parsedConfig.fileNames || [],
        ...tsconfig
      };
      
      this.cache.set(resolvedPath, configData);
      return configData;
    } catch (error) {
      console.warn(`Failed to parse tsconfig.json at ${resolvedPath}:`, error);
      const emptyConfig: TsConfigData = {};
      this.cache.set(resolvedPath, emptyConfig);
      return emptyConfig;
    }
  }

  /**
   * Get path alias configuration
   * @param projectRoot Project root directory
   * @param customPath Custom tsconfig.json path (optional)
   * @returns Path alias mapping
   */
  public getPathAliases(projectRoot: string, customPath?: string): Record<string, string[]> {
    const config = this.getTsConfig(projectRoot, customPath);
    const aliases: Record<string, string[]> = {};
    
    if (config.paths) {
      for (const [alias, targets] of Object.entries(config.paths)) {
        if (Array.isArray(targets)) {
          aliases[alias] = targets.map(target => 
            path.resolve(path.dirname(customPath || this.globalConfigPath || path.join(projectRoot, 'tsconfig.json')), target)
          );
        }
      }
    }
    
    return aliases;
  }

  /**
   * Clear all caches
   */
  public clearCache(): void {
    this.cache.clear();
  }
}