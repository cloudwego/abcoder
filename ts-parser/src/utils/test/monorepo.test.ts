import { describe, it, expect, jest } from '@jest/globals';
import * as fs from 'fs';
import * as path from 'path';
import { MonorepoUtils, EdenMonorepoConfig, MonorepoPackage } from '../monorepo';
import { 
  createEdenMonorepoProject, 
  createPnpmWorkspaceProject, 
  createLernaMonorepoProject,
  createEdenWorkspacesProject,
} from './test-utils';

describe('MonorepoUtils', () => {

  describe('isMonorepo', () => {
    it('should return true for Eden monorepo', () => {
      const testProject = createEdenMonorepoProject([]);
      expect(MonorepoUtils.isMonorepo(testProject.rootDir)).toBe(true);
      testProject.cleanup();
    });

    it('should return true for pnpm workspace', () => {
      const testProject = createPnpmWorkspaceProject([]);
      expect(MonorepoUtils.isMonorepo(testProject.rootDir)).toBe(true);
      testProject.cleanup();
    });

    it('should return true for lerna monorepo', () => {
      const testProject = createLernaMonorepoProject([]);
      expect(MonorepoUtils.isMonorepo(testProject.rootDir)).toBe(true);
      testProject.cleanup();
    });

    it('should return false for non-monorepo directory', () => {
      const testProject = createEdenMonorepoProject([]);
      // Remove the eden.monorepo.json to make it a non-monorepo
      fs.unlinkSync(path.join(testProject.rootDir, 'eden.monorepo.json'));
      expect(MonorepoUtils.isMonorepo(testProject.rootDir)).toBe(false);
      testProject.cleanup();
    });
  });

  describe('detectMonorepoType', () => {
    it('should detect Eden monorepo type', () => {
      const testProject = createEdenMonorepoProject([]);
      const edenConfigPath = path.join(testProject.rootDir, 'eden.monorepo.json');

      const result = MonorepoUtils.detectMonorepoType(testProject.rootDir);
      expect(result).toEqual({
        type: 'eden',
        configPath: edenConfigPath
      });

      testProject.cleanup();
    });

    it('should detect pnpm workspace type', () => {
      const testProject = createPnpmWorkspaceProject([]);
      const pnpmWorkspacePath = path.join(testProject.rootDir, 'pnpm-workspace.yaml');

      const result = MonorepoUtils.detectMonorepoType(testProject.rootDir);
      expect(result).toEqual({
        type: 'pnpm',
        configPath: pnpmWorkspacePath
      });

      testProject.cleanup();
    });

    it('should detect lerna monorepo type', () => {
      const testProject = createLernaMonorepoProject([]);
      const lernaConfigPath = path.join(testProject.rootDir, 'lerna.json');

      const result = MonorepoUtils.detectMonorepoType(testProject.rootDir);
      expect(result).toEqual({
        type: 'lerna',
        configPath: lernaConfigPath
      });

      testProject.cleanup();
    });

    it('should return null for non-monorepo directory', () => {
      const testProject = createEdenMonorepoProject([]);
      // Remove the eden.monorepo.json to make it a non-monorepo
      fs.unlinkSync(path.join(testProject.rootDir, 'eden.monorepo.json'));

      const result = MonorepoUtils.detectMonorepoType(testProject.rootDir);
      expect(result).toBeNull();

      testProject.cleanup();
    });

    it('should prioritize Eden over other types', () => {
      const testProject = createEdenMonorepoProject([]);
      const edenConfigPath = path.join(testProject.rootDir, 'eden.monorepo.json');
      
      // Add pnpm-workspace.yaml to test priority
      const pnpmWorkspacePath = path.join(testProject.rootDir, 'pnpm-workspace.yaml');
      fs.writeFileSync(pnpmWorkspacePath, 'packages:\n  - "packages/*"');

      const result = MonorepoUtils.detectMonorepoType(testProject.rootDir);
      expect(result).toEqual({
        type: 'eden',
        configPath: edenConfigPath
      });

      testProject.cleanup();
    });
  });

  describe('parseEdenMonorepoConfig', () => {
    it('should parse valid Eden monorepo config', () => {
      const testProject = createEdenMonorepoProject([
        { path: 'packages/core', shouldPublish: true },
        { path: 'packages/utils' }
      ]);

      const configPath = path.join(testProject.rootDir, 'eden.monorepo.json');
      const result = MonorepoUtils.parseEdenMonorepoConfig(configPath);
      
      expect(result).toEqual({
        packages: [
          { path: 'packages/core', shouldPublish: true },
          { path: 'packages/utils', shouldPublish: false }
        ]
      });

      testProject.cleanup();
    });

    it('should return null for non-existent config file', () => {
      const testProject = createEdenMonorepoProject([]);
      const configPath = path.join(testProject.rootDir, 'non-existent.json');
      
      const result = MonorepoUtils.parseEdenMonorepoConfig(configPath);
      expect(result).toBeNull();

      testProject.cleanup();
    });

    it('should return null for invalid JSON', () => {
      const testProject = createEdenMonorepoProject([]);
      const configPath = path.join(testProject.rootDir, 'invalid.json');
      fs.writeFileSync(configPath, 'invalid json content');

      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});
      const result = MonorepoUtils.parseEdenMonorepoConfig(configPath);
      
      expect(result).toBeNull();
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Failed to parse Eden monorepo config'),
        expect.any(Error)
      );
      
      consoleSpy.mockRestore();
      testProject.cleanup();
    });

    it('should get packages from workspaces glob patterns', () => {
      const testProject = createEdenWorkspacesProject([
        { 
          path: 'apps/web',
          packageJson: {
            name: '@test/web',
            version: '1.0.0'
          }
        },
        { 
          path: 'apps/mobile',
          packageJson: {
            name: '@test/mobile',
            version: '1.0.0'
          }
        },
        { 
          path: 'packages/core',
          packageJson: {
            name: '@test/core',
            version: '1.0.0'
          }
        },
        { 
          path: 'packages/utils',
          packageJson: {
            name: '@test/utils',
            version: '1.0.0'
          }
        }
      ], [
        'apps/*',
        'packages/*'
      ]);

      const config: EdenMonorepoConfig = {
        workspaces: [
          'apps/*',
          'packages/*'
        ]
      };

      const result = MonorepoUtils.getEdenPackages(testProject.rootDir, config);
      
      expect(result).toHaveLength(4);
      
      // Check apps
      const webApp = result.find(pkg => pkg.name === '@test/web');
      expect(webApp).toBeDefined();
      expect(webApp?.path).toBe('apps/web');
      expect(webApp?.shouldPublish).toBe(false);
      
      const mobileApp = result.find(pkg => pkg.name === '@test/mobile');
      expect(mobileApp).toBeDefined();
      expect(mobileApp?.path).toBe('apps/mobile');
      
      // Check packages
      const corePackage = result.find(pkg => pkg.name === '@test/core');
      expect(corePackage).toBeDefined();
      expect(corePackage?.path).toBe('packages/core');
      
      const utilsPackage = result.find(pkg => pkg.name === '@test/utils');
      expect(utilsPackage).toBeDefined();
      expect(utilsPackage?.path).toBe('packages/utils');

      testProject.cleanup();
    });

    it('should handle nested workspace patterns', () => {
      const testProject = createEdenWorkspacesProject([
        { 
          path: 'packages/ulink/core',
          packageJson: {
            name: '@ulink/core',
            version: '1.0.0'
          }
        },
        { 
          path: 'packages/ulink/utils',
          packageJson: {
            name: '@ulink/utils',
            version: '1.0.0'
          }
        }
      ], [
        'packages/ulink/*'
      ]);

      const config: EdenMonorepoConfig = {
        workspaces: [
          'packages/ulink/*'
        ]
      };

      const result = MonorepoUtils.getEdenPackages(testProject.rootDir, config);
      
      expect(result).toHaveLength(2);
      expect(result[0].path).toBe('packages/ulink/core');
      expect(result[1].path).toBe('packages/ulink/utils');

      testProject.cleanup();
    });

    it('should handle exact workspace paths', () => {
      const testProject = createEdenWorkspacesProject([
        { 
          path: 'libs/shared',
          packageJson: {
            name: '@test/shared',
            version: '1.0.0'
          }
        }
      ], [
        'libs/shared'
      ]);

      const config: EdenMonorepoConfig = {
        workspaces: [
          'libs/shared'
        ]
      };

      const result = MonorepoUtils.getEdenPackages(testProject.rootDir, config);
      
      expect(result).toHaveLength(1);
      expect(result[0].path).toBe('libs/shared');
      expect(result[0].name).toBe('@test/shared');
      expect(result[0].shouldPublish).toBe(false);

      testProject.cleanup();
    });

    it('should combine packages and workspaces formats', () => {
      const testProject = createEdenWorkspacesProject([
        { 
          path: 'legacy/old-package',
          packageJson: {
            name: '@test/old-package',
            version: '1.0.0'
          }
        },
        { 
          path: 'apps/new-app',
          packageJson: {
            name: '@test/new-app',
            version: '1.0.0'
          }
        }
      ], [
        'apps/*'
      ]);

      const config: EdenMonorepoConfig = {
        packages: [
          { path: 'legacy/old-package', shouldPublish: true }
        ],
        workspaces: [
          'apps/*'
        ]
      };

      const result = MonorepoUtils.getEdenPackages(testProject.rootDir, config);
      
      expect(result).toHaveLength(2);
      
      const oldPackage = result.find(pkg => pkg.name === '@test/old-package');
      expect(oldPackage).toBeDefined();
      expect(oldPackage?.shouldPublish).toBe(true);
      
      const newApp = result.find(pkg => pkg.name === '@test/new-app');
      expect(newApp).toBeDefined();
      expect(newApp?.shouldPublish).toBe(false);

      testProject.cleanup();
    });

    it('should skip directories without package.json in workspaces', () => {
      const testProject = createEdenWorkspacesProject([
        { 
          path: 'packages/with-package-json',
          packageJson: {
            name: '@test/with-package-json',
            version: '1.0.0'
          }
        }
      ], [
        'packages/*'
      ]);

      // Create a directory without package.json
      const withoutPackageJsonDir = path.join(testProject.rootDir, 'packages', 'without-package-json');
      fs.mkdirSync(withoutPackageJsonDir, { recursive: true });

      const config: EdenMonorepoConfig = {
        workspaces: [
          'packages/*'
        ]
      };

      const result = MonorepoUtils.getEdenPackages(testProject.rootDir, config);
      
      expect(result).toHaveLength(1);
      expect(result[0].name).toBe('@test/with-package-json');

      testProject.cleanup();
    });

    it('should handle non-existent workspace base directories', () => {
      const testProject = createEdenWorkspacesProject([], []);

      const config: EdenMonorepoConfig = {
        workspaces: [
          'non-existent/*'
        ]
      };

      const result = MonorepoUtils.getEdenPackages(testProject.rootDir, config);
      
      expect(result).toHaveLength(0);

      testProject.cleanup();
    });

    it('should handle invalid package.json in workspace packages', () => {
      const testProject = createEdenWorkspacesProject([
        { 
          path: 'packages/valid',
          packageJson: {
            name: '@test/valid',
            version: '1.0.0'
          }
        }
      ], [
        'packages/*'
      ]);

      // Create a package with invalid package.json
      const invalidDir = path.join(testProject.rootDir, 'packages', 'invalid');
      fs.mkdirSync(invalidDir, { recursive: true });
      fs.writeFileSync(path.join(invalidDir, 'package.json'), 'invalid json');

      const config: EdenMonorepoConfig = {
        workspaces: [
          'packages/*'
        ]
      };

      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});
      const result = MonorepoUtils.getEdenPackages(testProject.rootDir, config);
      
      expect(result).toHaveLength(2);
      expect(result.find(pkg => pkg.name === '@test/valid')).toBeDefined();
      expect(result.find(pkg => pkg.path === 'packages/invalid')).toBeDefined();
      expect(result.find(pkg => pkg.path === 'packages/invalid')?.name).toBeUndefined();
      
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Failed to parse package.json'),
        expect.any(Error)
      );
      
      consoleSpy.mockRestore();
      testProject.cleanup();
    });

    it('should handle complex workspace patterns like in real Eden config', () => {
      const testProject = createEdenWorkspacesProject([
        { 
          path: 'apps/web',
          packageJson: { name: '@test/web', version: '1.0.0' }
        },
        { 
          path: 'packages/core',
          packageJson: { name: '@test/core', version: '1.0.0' }
        },
        { 
          path: 'packages/ulink/auth',
          packageJson: { name: '@ulink/auth', version: '1.0.0' }
        },
        { 
          path: 'packages/config/eslint/plugins/custom',
          packageJson: { name: '@config/eslint-plugin-custom', version: '1.0.0' }
        },
        { 
          path: 'libs/shared',
          packageJson: { name: '@test/shared', version: '1.0.0' }
        }
      ], [
        'apps/*',
        'packages/*',
        'packages/ulink/*',
        'packages/config/eslint/plugins/*',
        'libs/*'
      ]);

      const config: EdenMonorepoConfig = {
        $schema: 'https://sf-unpkg-src.bytedance.net/@ies/eden-monorepo@3.1.0/lib/monorepo.schema.json',
        config: {
          cache: false,
          infraDir: '',
          pnpmVersion: '9.14.4',
          edenMonoVersion: '3.5.0',
          scriptName: {
            test: ['test'],
            build: ['build'],
            start: ['build:watch', 'dev', 'start', 'serve']
          },
          pluginsDir: 'packages/plugins/emo',
          plugins: ['./packages/config/emo/kesong-build.ts', '@ulike/emo-plugin-ci', '@ulike/emo-plugin-lint-assist'],
          autoInstallDepsForPlugins: false
        },
        workspaces: [
          'apps/*',
          'packages/*',
          'packages/ulink/*',
          'packages/config/eslint/plugins/*',
          'libs/*'
        ]
      };

      const result = MonorepoUtils.getEdenPackages(testProject.rootDir, config);
      
      expect(result).toHaveLength(5);
      expect(result.find(pkg => pkg.name === '@test/web')).toBeDefined();
      expect(result.find(pkg => pkg.name === '@test/core')).toBeDefined();
      expect(result.find(pkg => pkg.name === '@ulink/auth')).toBeDefined();
      expect(result.find(pkg => pkg.name === '@config/eslint-plugin-custom')).toBeDefined();
      expect(result.find(pkg => pkg.name === '@test/shared')).toBeDefined();

      testProject.cleanup();
    });

    it('should handle workspace pattern expansion errors gracefully', () => {
       const testProject = createEdenWorkspacesProject([], []);

       const config: EdenMonorepoConfig = {
         workspaces: [
           'packages/*'
         ]
       };

       // Test with a workspace pattern that points to a non-directory file
       const packagesFile = path.join(testProject.rootDir, 'packages');
       fs.writeFileSync(packagesFile, 'not a directory');

       const consoleSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});
       const result = MonorepoUtils.getEdenPackages(testProject.rootDir, config);
       
       expect(result).toHaveLength(0);
       // The function should handle the error gracefully and return empty array
       
       consoleSpy.mockRestore();
       testProject.cleanup();
     });
  });

  describe('getEdenPackages', () => {
    it('should get packages from Eden config', () => {
      const testProject = createEdenMonorepoProject([
        { 
          path: 'packages/core', 
          shouldPublish: true,
          packageJson: {
            name: '@test/core',
            version: '1.0.0'
          }
        },
        { 
          path: 'packages/utils', 
          shouldPublish: false,
          packageJson: {
            name: '@test/utils',
            version: '1.0.0'
          }
        }
      ]);

      const config: EdenMonorepoConfig = {
        packages: [
          { path: 'packages/core', shouldPublish: true },
          { path: 'packages/utils', shouldPublish: false }
        ]
      };

      const result = MonorepoUtils.getEdenPackages(testProject.rootDir, config);
      
      expect(result).toHaveLength(2);
      expect(result[0]).toEqual({
        path: 'packages/core',
        absolutePath: path.join(testProject.rootDir, 'packages/core'),
        shouldPublish: true,
        name: '@test/core'
      });
      expect(result[1]).toEqual({
        path: 'packages/utils',
        absolutePath: path.join(testProject.rootDir, 'packages/utils'),
        shouldPublish: false,
        name: '@test/utils'
      });

      testProject.cleanup();
    });

    it('should handle packages without package.json', () => {
      const testProject = createEdenMonorepoProject([
        { path: 'packages/core', shouldPublish: true }
      ]);

      const config: EdenMonorepoConfig = {
        packages: [
          { path: 'packages/core', shouldPublish: true }
        ]
      };

      const result = MonorepoUtils.getEdenPackages(testProject.rootDir, config);
      
      expect(result).toHaveLength(1);
      expect(result[0]).toEqual({
        path: 'packages/core',
        absolutePath: path.join(testProject.rootDir, 'packages/core'),
        shouldPublish: true,
        name: undefined
      });

      testProject.cleanup();
    });

    it('should skip non-existent package directories', () => {
      const testProject = createEdenMonorepoProject([]);

      const config: EdenMonorepoConfig = {
        packages: [
          { path: 'packages/non-existent', shouldPublish: true }
        ]
      };

      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});
      const result = MonorepoUtils.getEdenPackages(testProject.rootDir, config);
      
      expect(result).toHaveLength(0);
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Package directory does not exist')
      );
      
      consoleSpy.mockRestore();
      testProject.cleanup();
    });

    it('should handle invalid package.json', () => {
      const testProject = createEdenMonorepoProject([
        { path: 'packages/core', shouldPublish: true }
      ]);

      // Write invalid JSON to the package.json file
      const coreDir = path.join(testProject.rootDir, 'packages', 'core');
      fs.writeFileSync(path.join(coreDir, 'package.json'), 'invalid json');

      const config: EdenMonorepoConfig = {
        packages: [
          { path: 'packages/core', shouldPublish: true }
        ]
      };

      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});
      const result = MonorepoUtils.getEdenPackages(testProject.rootDir, config);
      
      expect(result).toHaveLength(1);
      expect(result[0].name).toBeUndefined();
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Failed to parse package.json'),
        expect.any(Error)
      );
      
      consoleSpy.mockRestore();
      testProject.cleanup();
    });
  });

  describe('getMonorepoPackages', () => {
    it('should get packages from Eden monorepo', () => {
      const testProject = createEdenMonorepoProject([
        { 
          path: 'packages/core', 
          shouldPublish: true,
          packageJson: {
            name: '@test/core',
            version: '1.0.0'
          }
        }
      ]);

      const result = MonorepoUtils.getMonorepoPackages(testProject.rootDir);
      
      expect(result).toHaveLength(1);
      expect(result[0]).toEqual({
        path: 'packages/core',
        absolutePath: path.join(testProject.rootDir, 'packages/core'),
        shouldPublish: true,
        name: '@test/core'
      });

      testProject.cleanup();
    });

    it('should get packages from pnpm workspace', () => {
      const testProject = createPnpmWorkspaceProject([
        { 
          path: 'packages/core',
          packageJson: {
            name: '@test/core',
            version: '1.0.0'
          }
        }
      ]);

      const result = MonorepoUtils.getMonorepoPackages(testProject.rootDir);
      
      expect(result).toHaveLength(1);
      expect(result[0]).toEqual({
        path: 'packages/core',
        absolutePath: path.join(testProject.rootDir, 'packages/core'),
        shouldPublish: false,
        name: '@test/core'
      });

      testProject.cleanup();
    });

    it('should return empty array for non-monorepo', () => {
      const testProject = createEdenMonorepoProject([]);
      // Remove the eden.monorepo.json to make it a non-monorepo
      fs.unlinkSync(path.join(testProject.rootDir, 'eden.monorepo.json'));

      const result = MonorepoUtils.getMonorepoPackages(testProject.rootDir);
      expect(result).toEqual([]);

      testProject.cleanup();
    });

    it('should handle unsupported monorepo types', () => {
      const testProject = createLernaMonorepoProject([], { packages: [] });

      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});
      const result = MonorepoUtils.getMonorepoPackages(testProject.rootDir);
      
      expect(result).toEqual([]);
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining("Monorepo type 'lerna' is not yet supported")
      );
      
      consoleSpy.mockRestore();
      testProject.cleanup();
    });
  });

  describe('findPackageForPath', () => {
    const packages: MonorepoPackage[] = [
      {
        path: 'packages/core',
        absolutePath: '/test/packages/core',
        shouldPublish: true,
        name: '@test/core'
      },
      {
        path: 'packages/utils',
        absolutePath: '/test/packages/utils',
        shouldPublish: false,
        name: '@test/utils'
      }
    ];

    it('should find package for file within package directory', () => {
      const filePath = '/test/packages/core/src/index.ts';
      const result = MonorepoUtils.findPackageForPath(filePath, packages);
      expect(result).toEqual(packages[0]);
    });

    it('should find package for package root directory', () => {
      const filePath = '/test/packages/core';
      const result = MonorepoUtils.findPackageForPath(filePath, packages);
      expect(result).toEqual(packages[0]);
    });

    it('should return null for file outside package directories', () => {
      const filePath = '/test/other/file.ts';
      const result = MonorepoUtils.findPackageForPath(filePath, packages);
      expect(result).toBeNull();
    });

    it('should return null for empty packages array', () => {
      const filePath = '/test/packages/core/src/index.ts';
      const result = MonorepoUtils.findPackageForPath(filePath, []);
      expect(result).toBeNull();
    });
  });
});