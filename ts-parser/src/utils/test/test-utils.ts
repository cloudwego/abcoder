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

export interface MonorepoTestProject {
  rootDir: string;
  cleanup: () => void;
}

export function createEdenMonorepoProject(packages: Array<{
  path: string;
  shouldPublish?: boolean;
  packageJson?: any;
}>): MonorepoTestProject {
  const uniqueId = Date.now() + '_' + Math.random().toString(36).substring(2, 15);
  const rootDir = path.join(__dirname, 'temp', uniqueId);
  
  fs.mkdirSync(rootDir, { recursive: true });
  
  // Create Eden monorepo config
  const edenConfig = {
    packages: packages.map(pkg => ({
      path: pkg.path,
      shouldPublish: pkg.shouldPublish ?? false
    }))
  };
  fs.writeFileSync(path.join(rootDir, 'eden.monorepo.json'), JSON.stringify(edenConfig, null, 2));
  
  // Create package directories and package.json files
  packages.forEach(pkg => {
    const packageDir = path.join(rootDir, pkg.path);
    fs.mkdirSync(packageDir, { recursive: true });
    
    if (pkg.packageJson) {
      fs.writeFileSync(
        path.join(packageDir, 'package.json'),
        JSON.stringify(pkg.packageJson, null, 2)
      );
    }
  });
  
  const cleanup = () => {
    if (fs.existsSync(rootDir)) {
      fs.rmSync(rootDir, { recursive: true, force: true });
    }
  };
  
  return { rootDir, cleanup };
}

export function createPnpmWorkspaceProject(packages: Array<{
  path: string;
  packageJson?: any;
}>, workspaceConfig: string[] = ['packages/*']): MonorepoTestProject {
  const uniqueId = Date.now() + '_' + Math.random().toString(36).substring(2, 15);
  const rootDir = path.join(__dirname, 'temp', uniqueId);
  
  fs.mkdirSync(rootDir, { recursive: true });
  
  // Create pnpm-workspace.yaml
  const workspaceYaml = `packages:\n${workspaceConfig.map(pattern => `  - "${pattern}"`).join('\n')}`;
  fs.writeFileSync(path.join(rootDir, 'pnpm-workspace.yaml'), workspaceYaml);
  
  // Create package directories and package.json files
  packages.forEach(pkg => {
    const packageDir = path.join(rootDir, pkg.path);
    fs.mkdirSync(packageDir, { recursive: true });
    
    if (pkg.packageJson) {
      fs.writeFileSync(
        path.join(packageDir, 'package.json'),
        JSON.stringify(pkg.packageJson, null, 2)
      );
    }
  });
  
  const cleanup = () => {
    if (fs.existsSync(rootDir)) {
      fs.rmSync(rootDir, { recursive: true, force: true });
    }
  };
  
  return { rootDir, cleanup };
}

export function createLernaMonorepoProject(packages: Array<{
  path: string;
  packageJson?: any;
}>, lernaConfig: any = { packages: ['packages/*'] }): MonorepoTestProject {
  const uniqueId = Date.now() + '_' + Math.random().toString(36).substring(2, 15);
  const rootDir = path.join(__dirname, 'temp', uniqueId);
  
  fs.mkdirSync(rootDir, { recursive: true });
  
  // Create lerna.json
  fs.writeFileSync(path.join(rootDir, 'lerna.json'), JSON.stringify(lernaConfig, null, 2));
  
  // Create package directories and package.json files
  packages.forEach(pkg => {
    const packageDir = path.join(rootDir, pkg.path);
    fs.mkdirSync(packageDir, { recursive: true });
    
    if (pkg.packageJson) {
      fs.writeFileSync(
        path.join(packageDir, 'package.json'),
        JSON.stringify(pkg.packageJson, null, 2)
      );
    }
  });
  
  const cleanup = () => {
    if (fs.existsSync(rootDir)) {
      fs.rmSync(rootDir, { recursive: true, force: true });
    }
  };
  
  return { rootDir, cleanup };
}