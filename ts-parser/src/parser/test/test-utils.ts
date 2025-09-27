import { Project, SourceFile } from 'ts-morph';
import * as path from 'path';
import * as fs from 'fs';

export interface TestProject {
  project: Project;
  sourceFile: SourceFile;
  cleanup: () => void;
}

export function createTestProject(code: string, fileName: string = 'test.ts'): TestProject {
  const uniqueId = Date.now() + '_' + Math.random().toString(36).substring(2, 15);
  const tempDir = path.join(__dirname, 'temp', uniqueId);
  
  fs.mkdirSync(tempDir, { recursive: true });
  
  const filePath = path.join(tempDir, fileName);
  fs.writeFileSync(filePath, code);
  
  const project = new Project({
    compilerOptions: {
      target: 99,
      module: 1,
      allowJs: true,
      skipLibCheck: true
    }
  });
  
  const sourceFile = project.addSourceFileAtPath(filePath);
  
  const cleanup = () => {
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  };
  
  return { project, sourceFile, cleanup };
}

export function createTestProjectWithMultipleFiles(files: Record<string, string>): TestProject {
  const uniqueId = Date.now() + '_' + Math.random().toString(36).substring(2, 15);
  const tempDir = path.join(__dirname, 'temp', uniqueId);
  
  fs.mkdirSync(tempDir, { recursive: true });
  
  // Write all files
  for (const [fileName, code] of Object.entries(files)) {
    const filePath = path.join(tempDir, fileName);
    fs.writeFileSync(filePath, code);
  }
  
  const project = new Project({
    compilerOptions: {
      target: 99,
      module: 1,
      allowJs: true,
      skipLibCheck: true
    }
  });
  
  // Add all files to project
  for (const fileName of Object.keys(files)) {
    const filePath = path.join(tempDir, fileName);
    project.addSourceFileAtPath(filePath);
  }
  
  // Return the main source file (test.ts by default)
  const mainFileName = Object.keys(files).includes('test.ts') ? 'test.ts' : Object.keys(files)[0];
  const sourceFile = project.getSourceFileOrThrow(path.join(tempDir, mainFileName));
  
  const cleanup = () => {
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  };
  
  return { project, sourceFile, cleanup };
}

export function createTestProjectWithTsConfig(code: string, tsConfig: any, fileName: string = 'test.ts'): TestProject {
  const uniqueId = Date.now() + '_' + Math.random().toString(36).substring(2, 15);
  const tempDir = path.join(__dirname, 'temp', uniqueId);
  
  fs.mkdirSync(tempDir, { recursive: true });
  
  const tsConfigPath = path.join(tempDir, 'tsconfig.json');
  fs.writeFileSync(tsConfigPath, JSON.stringify(tsConfig, null, 2));
  
  const filePath = path.join(tempDir, fileName);
  fs.writeFileSync(filePath, code);
  
  const project = new Project({
    tsConfigFilePath: tsConfigPath
  });
  
  const sourceFile = project.addSourceFileAtPath(filePath);
  
  const cleanup = () => {
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  };
  
  return { project, sourceFile, cleanup };
}

export function expectToBeDefined<T>(value: T | undefined | null): T {
  if (value === undefined || value === null) {
    throw new Error('Expected value to be defined');
  }
  return value;
}

export function expectArrayToContain<T>(array: T[], predicate: (item: T) => boolean): T {
  const found = array.find(predicate);
  if (!found) {
    throw new Error('Expected array to contain matching item');
  }
  return found;
}