import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { ProjectFactory } from '../package-processor';
import { createTestProject } from './test-utils';
import * as path from 'path';
import * as fs from 'fs';

// Mock ts module
jest.mock('typescript', () => ({
  createProgram: jest.fn(),
  getDefaultCompilerOptions: jest.fn(() => ({
    target: 99, // ScriptTarget.Latest
    module: 99, // ModuleKind.ESNext
    strict: true,
    esModuleInterop: true,
    skipLibCheck: true,
    forceConsistentCasingInFileNames: true,
  })),
  ScriptTarget: {
    Latest: 99,
  },
  ModuleKind: {
    ESNext: 99,
  },
}));

describe('ProjectFactory', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('createProjectForSingleRepo', () => {
    it('should create a ts-morph Project for a single repository', () => {
      // Create a real test project using test-utils
      const testProject = createTestProject(`
        export function hello() {
          return 'Hello, World!';
        }
      `);

      try {
        const repoPath = path.dirname(testProject.sourceFile.getFilePath());
        const result = ProjectFactory.createProjectForSingleRepo(repoPath);

        expect(result).toBeDefined();
        expect(typeof result.getSourceFiles).toBe('function');
        expect(typeof result.getTypeChecker).toBe('function');
        expect(typeof result.addSourceFileAtPath).toBe('function');
      } finally {
        testProject.cleanup();
      }
    });
  });

  it('should create a ts-morph Project for a package without name', () => {
    // Create a simple project using test-utils
    const testProject = createTestProject(`
        export const version = '1.0.0';
        export function getVersion() {
          return version;
        }
      `);

    try {
      const packagePath = path.dirname(testProject.sourceFile.getFilePath());
      const result = ProjectFactory.createProjectForPackage(packagePath);

      expect(result).toBeDefined();
      expect(typeof result.getSourceFiles).toBe('function');
      expect(typeof result.getTypeChecker).toBe('function');
    } finally {
      testProject.cleanup();
    }
  });

  it('should return default project when no tsconfig is found', () => {
    // Create a temporary directory without tsconfig.json
    const uniqueId = Date.now() + '_' + Math.random().toString(36).substring(2, 15);
    const tempDir = path.join(__dirname, 'temp', uniqueId);

    try {
      fs.mkdirSync(tempDir, { recursive: true });
      const result = ProjectFactory.createProjectForPackage(tempDir);

      expect(result).toBeDefined();
      expect(typeof result.getSourceFiles).toBe('function');
      expect(typeof result.getTypeChecker).toBe('function');
    } finally {
      if (fs.existsSync(tempDir)) {
        fs.rmSync(tempDir, { recursive: true, force: true });
      }
    }
  });
});

describe('createDefaultProject', () => {
  it('should create a ts-morph Project with default configuration', () => {
    const result = ProjectFactory.createDefaultProject();

    expect(result).toBeDefined();
    expect(typeof result.getSourceFiles).toBe('function');
    expect(typeof result.getTypeChecker).toBe('function');
    expect(typeof result.addSourceFileAtPath).toBe('function');
  });

  it('should create project with proper compiler options', () => {
    const result = ProjectFactory.createDefaultProject();
    const compilerOptions = result.getCompilerOptions();

    expect(compilerOptions.target).toBe(99); // ESNext
    expect(compilerOptions.allowJs).toBe(true);
    expect(compilerOptions.skipLibCheck).toBe(true);
    expect(compilerOptions.forceConsistentCasingInFileNames).toBe(true);
  });

  it('should create project that can handle source files', () => {
    const result = ProjectFactory.createDefaultProject();
    const sourceFiles = result.getSourceFiles();

    expect(Array.isArray(sourceFiles)).toBe(true);
    expect(sourceFiles.length).toBe(0); // Initially empty
  });
});

describe('error handling', () => {
  it('should handle invalid paths gracefully', () => {
    // Test with empty path - should return default project
    const result = ProjectFactory.createProjectForSingleRepo('');
    expect(result).toBeDefined();
    expect(typeof result.getSourceFiles).toBe('function');
    expect(typeof result.getTypeChecker).toBe('function');
  });

  it('should handle non-existent paths', () => {
    // Test with non-existent path - should return default project
    const result = ProjectFactory.createProjectForSingleRepo('/non/existent/path');
    expect(result).toBeDefined();
    expect(typeof result.getSourceFiles).toBe('function');
    expect(typeof result.getTypeChecker).toBe('function');
  });
});
