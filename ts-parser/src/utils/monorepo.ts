import * as fs from 'fs';
import * as path from 'path';

/**
 * Interface for Eden monorepo configuration
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
    cache?: boolean;
    workspaceCheck?: {
      dependencyVersionCheck?: {
        forceCheck?: boolean;
        autofix?: boolean;
      };
    };
    autoInstallDepsForPlugins?: boolean;
    plugins?: string[];
    scriptName?: {
      start?: string[];
    };
  };
  packages: Array<{
    path: string;
    shouldPublish?: boolean;
  }>;
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
   */
  static isMonorepo(rootPath: string): boolean {
    const edenConfigPath = path.join(rootPath, 'eden.monorepo.json');
    const pnpmWorkspacePath = path.join(rootPath, 'pnpm-workspace.yaml');
    const lernaConfigPath = path.join(rootPath, 'lerna.json');
    
    return fs.existsSync(edenConfigPath) || 
           fs.existsSync(pnpmWorkspacePath) || 
           fs.existsSync(lernaConfigPath);
  }

  /**
   * Detect monorepo type and return configuration file path
   */
  static detectMonorepoType(rootPath: string): { type: string; configPath: string } | null {
    const edenConfigPath = path.join(rootPath, 'eden.monorepo.json');
    const pnpmWorkspacePath = path.join(rootPath, 'pnpm-workspace.yaml');
    // const yarnWorkspacePath = path.join(rootPath, 'yarn.lock');
    const lernaConfigPath = path.join(rootPath, 'lerna.json');
    
    if (fs.existsSync(edenConfigPath)) {
      return { type: 'eden', configPath: edenConfigPath };
    }
    if (fs.existsSync(pnpmWorkspacePath)) {
      return { type: 'pnpm', configPath: pnpmWorkspacePath };
    }
    // if (fs.existsSync(yarnWorkspacePath)) {
    //   return { type: 'yarn', configPath: yarnWorkspacePath };
    // }
    if (fs.existsSync(lernaConfigPath)) {
      return { type: 'lerna', configPath: lernaConfigPath };
    }
    
    return null;
  }

  /**
   * Parse Eden monorepo configuration
   */
  static parseEdenMonorepoConfig(configPath: string): EdenMonorepoConfig | null {
    try {
      if (!fs.existsSync(configPath)) {
        return null;
      }
      
      const configContent = fs.readFileSync(configPath, 'utf-8');
      const config: EdenMonorepoConfig = JSON.parse(configContent);
      
      return config;
    } catch (error) {
      console.warn(`Failed to parse Eden monorepo config at ${configPath}:`, error);
      return null;
    }
  }

  /**
   * Get packages from Eden monorepo configuration
   */
  static getEdenPackages(rootPath: string, config: EdenMonorepoConfig): MonorepoPackage[] {
    const packages: MonorepoPackage[] = [];
    
    for (const pkg of config.packages) {
      const absolutePath = path.resolve(rootPath, pkg.path);
      
      // Check if package directory exists
      if (fs.existsSync(absolutePath)) {
        // Try to get package name from package.json
        let packageName: string | undefined;
        const packageJsonPath = path.join(absolutePath, 'package.json');
        
        if (fs.existsSync(packageJsonPath)) {
          try {
            const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf-8'));
            packageName = packageJson.name;
          } catch (error) {
            console.warn(`Failed to parse package.json at ${packageJsonPath}:`, error);
          }
        }
        
        packages.push({
          path: pkg.path,
          absolutePath,
          shouldPublish: pkg.shouldPublish ?? false,
          name: packageName
        });
      } else {
        console.warn(`Package directory does not exist: ${absolutePath}`);
      }
    }
    
    return packages;
  }

  /**
   * Get all packages from a monorepo
   */
  static getMonorepoPackages(rootPath: string): MonorepoPackage[] {
    const monorepoInfo = this.detectMonorepoType(rootPath);
    
    if (!monorepoInfo) {
      return [];
    }
    
    switch (monorepoInfo.type) {
      case 'eden': {
        const config = this.parseEdenMonorepoConfig(monorepoInfo.configPath);
        if (config) {
          return this.getEdenPackages(rootPath, config);
        }
        break;
      }
      case 'pnpm': {
        const configContent = fs.readFileSync(monorepoInfo.configPath, 'utf-8');
        const packages: MonorepoPackage[] = [];
        const lines = configContent.split('\n');
        let inPackages = false;
        for (const line of lines) {
          if (line.startsWith('packages:')) {
            inPackages = true;
            continue;
          }
          if (inPackages && line.trim().startsWith('-')) {
            const glob = line.trim().substring(1).trim().replace(/'/g, '').replace(/"/g, '');
            if (glob.endsWith('/*')) {
              const packageDir = path.join(rootPath, glob.slice(0, -2));
              if (fs.existsSync(packageDir) && fs.statSync(packageDir).isDirectory()) {
                const packageNames = fs.readdirSync(packageDir);
                for (const pkgName of packageNames) {
                  const pkgAbsolutePath = path.join(packageDir, pkgName);
                  if (fs.statSync(pkgAbsolutePath).isDirectory()) {
                    const pkgRelativePath = path.relative(rootPath, pkgAbsolutePath);
                    let packageName: string | undefined;
                    const packageJsonPath = path.join(pkgAbsolutePath, 'package.json');
                    if (fs.existsSync(packageJsonPath)) {
                      try {
                        const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf-8'));
                        packageName = packageJson.name;
                      } catch (error) {
                        console.warn(`Failed to parse package.json at ${packageJsonPath}:`, error);
                      }
                    }
                    packages.push({
                      path: pkgRelativePath,
                      absolutePath: pkgAbsolutePath,
                      shouldPublish: false, // Cannot determine from pnpm-workspace.yaml
                      name: packageName
                    });
                  }
                }
              }
            }
          } else if (inPackages && !/^\s*$/.test(line) && !/^\s+-/.test(line)) {
            // We are out of the packages section if the line is not empty and not a package entry
            break;
          }
        }
        return packages;
      }
      // TODO: Add support for other monorepo types (yarn, lerna)
      default:
        console.warn(`Monorepo type '${monorepoInfo.type}' is not yet supported`);
        break;
    }
    
    return [];
  }

  /**
   * Check if a path is within any of the monorepo packages
   */
  static findPackageForPath(filePath: string, packages: MonorepoPackage[]): MonorepoPackage | null {
    const absoluteFilePath = path.resolve(filePath);
    
    for (const pkg of packages) {
      if (absoluteFilePath.startsWith(pkg.absolutePath + path.sep) || 
          absoluteFilePath === pkg.absolutePath) {
        return pkg;
      }
    }
    
    return null;
  }
}