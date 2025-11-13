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
      const testProject = createEdenMonorepoProject([
        { 
          path: 'packages/core',
          packageJson: { name: '@test/core', version: '1.0.0' }
        },
        { 
          path: 'packages/utils',
          packageJson: { name: '@test/utils', version: '1.0.0' }
        }
      ]);
      expect(MonorepoUtils.isMonorepo(testProject.rootDir)).toBe(true);
      testProject.cleanup();
    });

    it('should return true for pnpm workspace', () => {
      const testProject = createPnpmWorkspaceProject([
        { 
          path: 'packages/core',
          packageJson: { name: '@test/core', version: '1.0.0' }
        },
        { 
          path: 'packages/utils',
          packageJson: { name: '@test/utils', version: '1.0.0' }
        }
      ]);
      expect(MonorepoUtils.isMonorepo(testProject.rootDir)).toBe(true);
      testProject.cleanup();
    });

    it('should return true for lerna monorepo', () => {
      const testProject = createLernaMonorepoProject([
        { 
          path: 'packages/core',
          packageJson: { name: '@test/core', version: '1.0.0' }
        },
        { 
          path: 'packages/utils',
          packageJson: { name: '@test/utils', version: '1.0.0' }
        }
      ]);
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



  describe('getMonorepoPackages', () => {
    it('should get packages via generic discovery from Eden monorepo', () => {
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
        shouldPublish: false, // shouldPublish is always false now
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

    it('should handle all monorepo types via generic discovery', () => {
      const testProject = createLernaMonorepoProject([
        { 
          path: 'packages/core',
          packageJson: {
            name: '@test/core',
            version: '1.0.0'
          }
        }
      ], { packages: [] });

      const result = MonorepoUtils.getMonorepoPackages(testProject.rootDir);
      
      // All monorepo types now use generic discovery
      expect(result).toHaveLength(1);
      expect(result[0].name).toBe('@test/core');
      
      testProject.cleanup();
    });
  });


});