import { describe, it, expect, jest } from '@jest/globals';
import * as fs from 'fs';
import * as path from 'path';
import { MonorepoUtils, EdenMonorepoConfig, MonorepoPackage } from '../monorepo';
import { 
  createEdenMonorepoProject, 
  createPnpmWorkspaceProject, 
  createLernaMonorepoProject,
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