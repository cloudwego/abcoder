import * as fs from 'fs';
import * as path from 'path';

/**
 * Interface for Eden monorepo pnpmWorkspace configuration
 * Available when emo version >= 3.6.0
 */
export interface PnpmWorkspaceConfig {
  // Workspace packages configuration (same as workspaces)
  packages?: string[];
}

/**
 * Interface for Eden monorepo configuration
 * Supports both legacy packages format and new workspaces format
 */
export interface EdenMonorepoConfig {
  $schema?: string;
  config?: {
    infraDir?: string;
    pnpmVersion?: string;
    edenMonoVersion?: string;
    pkgJsonDepsPolicies?: string;
    packagePublish?: {
      tool?: string;
    };
    cache?: boolean | object;
    workspaceCheck?: {
      dependencyVersionCheck?: {
        forceCheck?: boolean;
        autofix?: boolean;
      };
    };
    autoInstallDepsForPlugins?: boolean;
    plugins?: string[];
    pluginsDir?: string;
    scriptName?: {
      [key: string]: string[];
    };
    outputPaths?: {
      dirs?: string[];
      files?: string[];
    };
  };
  // Legacy packages format (optional for backward compatibility)
  packages?: Array<{
    path: string;
    shouldPublish?: boolean;
  }>;
  // New workspaces format (supports glob patterns)
  workspaces?: string[];
  // This has higher priority than packages and workspaces
  pnpmWorkspace?: PnpmWorkspaceConfig;
}

/**
 * Interface for package information
 */
export interface MonorepoPackage {
  path: string;
  absolutePath: string;
  shouldPublish: boolean;
  name?: string;
}

/**
 * Utility class for handling monorepo detection and configuration
 */
export class MonorepoUtils {
  /**
   * Check if a directory contains a monorepo configuration
   * Now determines monorepo by counting package.json files (>= 2 means monorepo)
   */
  static isMonorepo(rootPath: string): boolean {
    const packageJsonCount = this.countPackageJsonFiles(rootPath);
    return packageJsonCount >= 2;
  }

  /**
   * Count the number of package.json files in the directory tree
   */
  private static countPackageJsonFiles(rootPath: string): number {
    try {
      let count = 0;
      const items = fs.readdirSync(rootPath);
      
      for (const item of items) {
        const fullPath = path.join(rootPath, item);
        const stat = fs.statSync(fullPath);
        
        if (stat.isDirectory()) {
          // Skip node_modules and hidden directories
          if (item === 'node_modules' || item.startsWith('.')) {
            continue;
          }
          
          // Check if this directory has a package.json
          const packageJsonPath = path.join(fullPath, 'package.json');
          if (fs.existsSync(packageJsonPath)) {
            count++;
          }
          
          // Recursively count in subdirectories
          count += this.countPackageJsonFiles(fullPath);
        }
      }
      
      return count;
    } catch (error) {
      console.warn(`Error counting package.json files in ${rootPath}:`, error);
      return 0;
    }
  }

  /**
   * Get all packages from a monorepo
   * Unified approach: discover packages by package.json files only
   */
  static getMonorepoPackages(rootPath: string): MonorepoPackage[] {
    // Simply use generic package discovery, ignoring monorepo type
    return this.getGenericPackages(rootPath);
  }



  /**
   * Get packages by discovering package.json files (generic approach)
   */
  private static getGenericPackages(rootPath: string): MonorepoPackage[] {
    const packages: MonorepoPackage[] = [];
    this.discoverPackagesRecursive(rootPath, rootPath, packages);
    return packages;
  }

  /**
   * Recursively discover packages by finding package.json files
   */
  private static discoverPackagesRecursive(rootPath: string, currentDir: string, packages: MonorepoPackage[]): void {
    try {
      const items = fs.readdirSync(currentDir, { withFileTypes: true });
      
      for (const item of items) {
        if (!item.isDirectory()) {
          continue;
        }
        
        const dirName = item.name;
        const fullPath = path.join(currentDir, dirName);
        
        // Skip node_modules and hidden directories
        if (dirName === 'node_modules' || dirName.startsWith('.')) {
          continue;
        }
        
        // Check if this directory has a package.json
        const packageJsonPath = path.join(fullPath, 'package.json');
        if (fs.existsSync(packageJsonPath)) {
          try {
            const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf-8'));
            const relativePath = path.relative(rootPath, fullPath);
            
            packages.push({
              path: relativePath,
              absolutePath: fullPath,
              shouldPublish: false, // Default to false for generic discovery
              name: packageJson.name
            });
          } catch (error) {
            console.warn(`Failed to parse package.json at ${packageJsonPath}:`, error);
          }
        }
        
        // Recursively search in subdirectories
        this.discoverPackagesRecursive(rootPath, fullPath, packages);
      }
    } catch (error) {
      console.warn(`Error discovering packages in ${currentDir}:`, error);
    }
  }

}