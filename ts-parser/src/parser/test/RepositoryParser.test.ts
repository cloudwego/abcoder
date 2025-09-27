import { describe, it, expect, jest } from '@jest/globals';
import path from 'path';
import * as fs from 'fs';
import { RepositoryParser } from '../RepositoryParser';
import { MonorepoUtils } from '../../utils/monorepo';
import { ModuleParser } from '../ModuleParser';
import { GraphBuilder } from '../../utils/graph-builder';
import { createTestProject } from './test-utils';
import { 
  createEdenMonorepoProject, 
  createPnpmWorkspaceProject, 
} from '../../utils/test/test-utils';

describe('RepositoryParser', () => {
  // Basic functionality tests
  describe('Basic Functionality', () => {
    describe('constructor', () => {
      it('should create instance with project root', () => {
        const parser = new RepositoryParser('/test/project');
        expect(parser).toBeDefined();
      });

      it('should create instance with project root and tsconfig path', () => {
        const parser = new RepositoryParser('/test/project', '/test/tsconfig.json');
        expect(parser).toBeDefined();
      });
    });
  });

  // Eden Monorepo specific tests
  describe('Eden Monorepo Support', () => {
    it('should detect and parse Eden monorepo configuration', async () => {
      // Create Eden monorepo structure using template function
      const testProject = createEdenMonorepoProject([
        {
          path: 'packages/shared-utils',
          shouldPublish: false,
          packageJson: {
            "name": "@test/shared-utils",
            "version": "1.0.0",
            "main": "dist/index.js",
            "types": "dist/index.d.ts"
          }
        },
        {
          path: 'packages/api-server',
          shouldPublish: false,
          packageJson: {
            "name": "@test/api-server",
            "version": "1.0.0",
            "main": "dist/index.js",
            "types": "dist/index.d.ts"
          }
        }
      ]);

      // Create package structure
      const sharedUtilsDir = path.join(testProject.rootDir, 'packages', 'shared-utils', 'src');
      fs.mkdirSync(sharedUtilsDir, { recursive: true });

      // Create shared-utils TypeScript files
      const stringUtilsCode = `
/**
 * Formats a message with a prefix
 * @param message - The message to format
 * @param prefix - Optional prefix to add
 * @returns Formatted message
 */
export function formatMessage(message: string, prefix = ''): string {
  return \`\${prefix} \${message}\`
}

/**
 * Capitalizes the first letter of a string
 * @param str - The string to capitalize
 * @returns Capitalized string
 */
export function capitalize(str: string): string {
  if (!str) {
    return str
  }
  return str.charAt(0).toUpperCase() + str.slice(1)
}

export interface StringOptions {
  caseSensitive?: boolean;
  trimWhitespace?: boolean;
}
`;

      const indexCode = `
export * from './string-utils'
export * from './date-utils'

export const SHARED_CONSTANTS = {
  VERSION: '1.0.0',
  API_BASE_URL: 'https://api.example.com'
} as const;
`;

      const dateUtilsCode = `
export function formatDate(date: Date): string {
  return date.toISOString().split('T')[0];
}

export function addDays(date: Date, days: number): Date {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return result;
}
`;

      fs.writeFileSync(path.join(sharedUtilsDir, 'string-utils.ts'), stringUtilsCode);
      fs.writeFileSync(path.join(sharedUtilsDir, 'index.ts'), indexCode);
      fs.writeFileSync(path.join(sharedUtilsDir, 'date-utils.ts'), dateUtilsCode);

      // Create tsconfig for shared-utils
      const tsConfig = {
        "compilerOptions": {
          "target": "ES2020",
          "module": "ESNext",
          "moduleResolution": "node",
          "strict": true,
          "esModuleInterop": true,
          "skipLibCheck": true,
          "forceConsistentCasingInFileNames": true,
          "declaration": true,
          "outDir": "dist"
        },
        "include": ["src/**/*"],
        "exclude": ["node_modules", "dist"]
      };
      fs.writeFileSync(
        path.join(testProject.rootDir, 'packages', 'shared-utils', 'tsconfig.json'),
        JSON.stringify(tsConfig, null, 2)
      );

      // Create API server package
      const apiServerDir = path.join(testProject.rootDir, 'packages', 'api-server', 'src');
      fs.mkdirSync(apiServerDir, { recursive: true });

      const apiServerCode = `
import { formatMessage, capitalize } from '@test/shared-utils';

export class ApiServer {
  private port: number;

  constructor(port: number = 3000) {
    this.port = port;
  }

  start(): void {
    console.log(formatMessage(\`Server starting on port \${this.port}\`));
  }

  formatResponse(data: any): string {
    return capitalize(JSON.stringify(data));
  }
}

export interface ServerConfig {
  port: number;
  host: string;
  ssl?: boolean;
}
`;

      fs.writeFileSync(path.join(apiServerDir, 'index.ts'), apiServerCode);

      // Create API server package.json
      const apiServerPackageJson = {
        "name": "@test/api-server",
        "version": "1.0.0",
        "dependencies": {
          "@test/shared-utils": "workspace:*"
        }
      };
      fs.writeFileSync(
        path.join(testProject.rootDir, 'packages', 'api-server', 'package.json'),
        JSON.stringify(apiServerPackageJson, null, 2)
      );

      // Create API server tsconfig.json
      const apiServerTsConfig = {
        "compilerOptions": {
          "target": "ES2020",
          "module": "ESNext",
          "moduleResolution": "node",
          "strict": true,
          "esModuleInterop": true,
          "skipLibCheck": true,
          "forceConsistentCasingInFileNames": true,
          "declaration": true,
          "outDir": "dist",
          "baseUrl": ".",
          "paths": {
            "@test/*": ["../*/src"]
          }
        },
        "include": ["src/**/*"],
        "exclude": ["node_modules", "dist"]
      };
      fs.writeFileSync(
        path.join(testProject.rootDir, 'packages', 'api-server', 'tsconfig.json'),
        JSON.stringify(apiServerTsConfig, null, 2)
      );

      // Create root tsconfig
      const rootTsConfig = {
        "compilerOptions": {
          "target": "ES2020",
          "module": "ESNext",
          "moduleResolution": "node",
          "strict": true,
          "esModuleInterop": true,
          "skipLibCheck": true,
          "forceConsistentCasingInFileNames": true,
          "baseUrl": ".",
          "paths": {
            "@test/*": ["packages/*/src"]
          }
        },
        "references": [
          { "path": "./packages/shared-utils" },
          { "path": "./packages/api-server" }
        ]
      };
      fs.writeFileSync(
        path.join(testProject.rootDir, 'tsconfig.json'),
        JSON.stringify(rootTsConfig, null, 2)
      );

      // Parse the repository
      const parser = new RepositoryParser(testProject.rootDir);
      const result = await parser.parseRepository(testProject.rootDir);

      // Verify the result
      expect(result).toBeDefined();
      expect(result.id).toBeDefined();
      expect(result.ASTVersion).toBeDefined();

      // Verify modules (packages) are detected
      expect(result.Modules).toBeDefined();
      expect(Object.keys(result.Modules)).toContain('@test/shared-utils');
      expect(Object.keys(result.Modules)).toContain('@test/api-server');

      // Verify shared-utils module
      const sharedUtilsModule = result.Modules['@test/shared-utils'];
      expect(sharedUtilsModule).toBeDefined();
      expect(sharedUtilsModule.Packages).toBeDefined();
      expect(sharedUtilsModule.Packages.src).toBeDefined();

      // Verify functions are parsed
      const sharedUtilsFunctions = sharedUtilsModule.Packages.src.Functions;
      expect(sharedUtilsFunctions).toBeDefined();
      expect(sharedUtilsFunctions['formatMessage']).toBeDefined();
      expect(sharedUtilsFunctions['capitalize']).toBeDefined();
      expect(sharedUtilsFunctions['formatDate']).toBeDefined();
      expect(sharedUtilsFunctions['addDays']).toBeDefined();

      // Verify types are parsed
      const sharedUtilsTypes = sharedUtilsModule.Packages.src.Types;
      expect(sharedUtilsTypes).toBeDefined();
      expect(sharedUtilsTypes['StringOptions']).toBeDefined();

      // Verify variables are parsed
      const sharedUtilsVars = sharedUtilsModule.Packages.src.Vars;
      expect(sharedUtilsVars).toBeDefined();
      expect(sharedUtilsVars['SHARED_CONSTANTS']).toBeDefined();

      // Verify api-server module
      const apiServerModule = result.Modules['@test/api-server'];
      expect(apiServerModule).toBeDefined();
      expect(apiServerModule.Packages.src.Types['ApiServer']).toBeDefined();
      expect(apiServerModule.Packages.src.Types['ServerConfig']).toBeDefined();

      // Verify dependency graph includes cross-package dependencies
      expect(result.Graph).toBeDefined();
      const graphKeys = Object.keys(result.Graph);
      expect(graphKeys.length).toBeGreaterThan(0);

      // Check that the graph contains references to shared utilities
      const hasSharedUtilsRefs = graphKeys.some(key => 
        key.includes('formatMessage') || key.includes('capitalize')
      );
      expect(hasSharedUtilsRefs).toBe(true);

      // Cleanup
      testProject.cleanup();
    });

    it('should handle Eden monorepo with complex package dependencies', async () => {
      // Create a more complex Eden monorepo structure using template function
      const testProject = createEdenMonorepoProject([
        {
          path: 'packages/core',
          shouldPublish: true,
          packageJson: {
            "name": "@test/core",
            "version": "1.0.0",
            "main": "dist/index.js",
            "types": "dist/index.d.ts"
          }
        },
        {
          path: 'packages/ui-components',
          shouldPublish: true,
          packageJson: {
            "name": "@test/ui-components",
            "version": "1.0.0",
            "main": "dist/index.js",
            "types": "dist/index.d.ts",
            "dependencies": {
              "@test/core": "workspace:*"
            }
          }
        },
        {
          path: 'apps/web-app',
          shouldPublish: false,
          packageJson: {
            "name": "@test/web-app",
            "version": "1.0.0",
            "private": true,
            "dependencies": {
              "@test/core": "workspace:*",
              "@test/ui-components": "workspace:*"
            }
          }
        }
      ]);

      // Create core package
      const coreDir = path.join(testProject.rootDir, 'packages', 'core', 'src');
      fs.mkdirSync(coreDir, { recursive: true });

      const coreCode = `
export abstract class BaseService {
  protected abstract serviceName: string;
  
  abstract initialize(): Promise<void>;
  
  getServiceInfo(): { name: string; version: string } {
    return {
      name: this.serviceName,
      version: '1.0.0'
    };
  }
}

export interface ServiceConfig {
  timeout: number;
  retries: number;
}

export type ServiceStatus = 'idle' | 'running' | 'error';
`;

      fs.writeFileSync(path.join(coreDir, 'index.ts'), coreCode);

      // Create UI components package
      const uiDir = path.join(testProject.rootDir, 'packages', 'ui-components', 'src');
      fs.mkdirSync(uiDir, { recursive: true });

      const uiCode = `
import { BaseService, ServiceConfig } from '@test/core';

export class UIService extends BaseService {
  protected serviceName = 'UIService';
  
  async initialize(): Promise<void> {
    console.log('Initializing UI Service');
  }
  
  renderComponent(config: ServiceConfig): string {
    return \`<div>Component with timeout: \${config.timeout}</div>\`;
  }
}

export interface ComponentProps {
  title: string;
  visible: boolean;
}
`;

      fs.writeFileSync(path.join(uiDir, 'index.ts'), uiCode);

      // Create web app
      const webAppDir = path.join(testProject.rootDir, 'apps', 'web-app', 'src');
      fs.mkdirSync(webAppDir, { recursive: true });

      const webAppCode = `
import { UIService } from '@test/ui-components';
import { ServiceConfig } from '@test/core';

export class WebApplication {
  private uiService: UIService;
  
  constructor() {
    this.uiService = new UIService();
  }
  
  async start(): Promise<void> {
    await this.uiService.initialize();
    
    const config: ServiceConfig = {
      timeout: 5000,
      retries: 3
    };
    
    const html = this.uiService.renderComponent(config);
    console.log(html);
  }
}
`;

      fs.writeFileSync(path.join(webAppDir, 'index.ts'), webAppCode);

      // Create package.json files
      const packages = [
        { name: 'core', path: 'packages/core' },
        { name: 'ui-components', path: 'packages/ui-components' },
        { name: 'web-app', path: 'apps/web-app' }
      ];

      packages.forEach(pkg => {
        const packageJson = {
          name: `@test/${pkg.name}`,
          version: "1.0.0",
          main: "dist/index.js",
          types: "dist/index.d.ts",
          dependencies: pkg.name === 'ui-components' ? { "@test/core": "workspace:*" } :
                       pkg.name === 'web-app' ? { "@test/core": "workspace:*", "@test/ui-components": "workspace:*" } : {}
        };
        
        fs.writeFileSync(
          path.join(testProject.rootDir, pkg.path, 'package.json'),
          JSON.stringify(packageJson, null, 2)
        );

        // Create tsconfig.json for each package
        const packageTsConfig = {
          "compilerOptions": {
            "target": "ES2020",
            "module": "ESNext",
            "moduleResolution": "node",
            "strict": true,
            "esModuleInterop": true,
            "skipLibCheck": true,
            "forceConsistentCasingInFileNames": true,
            "declaration": true,
            "outDir": "dist",
            "baseUrl": ".",
            "paths": {
              "@test/*": ["../*/src", "../../packages/*/src"]
            }
          },
          "include": ["src/**/*"],
          "exclude": ["node_modules", "dist"]
        };
        
        fs.writeFileSync(
          path.join(testProject.rootDir, pkg.path, 'tsconfig.json'),
          JSON.stringify(packageTsConfig, null, 2)
        );
      });

      // Parse the repository
      const parser = new RepositoryParser(testProject.rootDir);
      const result = await parser.parseRepository(testProject.rootDir);

      // Verify all modules are detected
      expect(Object.keys(result.Modules)).toContain('@test/core');
      expect(Object.keys(result.Modules)).toContain('@test/ui-components');
      expect(Object.keys(result.Modules)).toContain('@test/web-app');

      // Verify inheritance is captured
      const uiModule = result.Modules['@test/ui-components'];
      const uiServiceType = uiModule.Packages.src.Types['UIService'];
      expect(uiServiceType).toBeDefined();
      expect(uiServiceType.Exported).toBe(true);

      // Verify cross-package dependencies in graph
      const graphKeys = Object.keys(result.Graph);
      const hasCoreDependency = graphKeys.some(key => 
        key.includes('BaseService') || key.includes('ServiceConfig')
      );
      expect(hasCoreDependency).toBe(true);
      
      await testProject.cleanup();
    });
  });

  // Integration tests for module collaboration
  describe('Integration Tests - Module Collaboration', () => {
    it('should integrate MonorepoUtils, ModuleParser, and GraphBuilder correctly', async () => {
       // Create a test monorepo structure using PNPM workspace
       const testProject = createPnpmWorkspaceProject([
         {
           path: 'packages/core',
           packageJson: {
             "name": "@test/core",
             "version": "1.0.0",
             "main": "dist/index.js",
             "types": "dist/index.d.ts"
           }
         },
         {
           path: 'packages/ui',
           packageJson: {
             "name": "@test/ui",
             "version": "1.0.0",
             "main": "dist/index.js",
             "types": "dist/index.d.ts",
             "dependencies": {
               "@test/core": "1.0.0"
             }
           }
         }
       ]);

       // Create root package.json
       const rootPackageJson = {
         "name": "test-monorepo",
         "private": true,
         "devDependencies": {
           "typescript": "^5.0.0"
         }
       };
       fs.writeFileSync(path.join(testProject.rootDir, 'package.json'), JSON.stringify(rootPackageJson, null, 2));

      // Create packages source directories
      const packageADir = path.join(testProject.rootDir, 'packages', 'core');
      fs.mkdirSync(path.join(packageADir, 'src'), { recursive: true });
      
      const coreCode = `
export interface Config {
  apiUrl: string;
  timeout: number;
}

export class BaseService {
  protected config: Config;
  
  constructor(config: Config) {
    this.config = config;
  }
  
  protected async makeRequest(endpoint: string): Promise<any> {
    // Implementation
    return {};
  }
}

export function createConfig(apiUrl: string): Config {
  return {
    apiUrl,
    timeout: 5000
  };
}
`;
      fs.writeFileSync(path.join(packageADir, 'src', 'index.ts'), coreCode);

      // Package B - UI components that depend on core
      const packageBDir = path.join(testProject.rootDir, 'packages', 'ui');
      fs.mkdirSync(path.join(packageBDir, 'src'), { recursive: true });
      
      const packageBJson = {
        "name": "@test/ui",
        "version": "1.0.0",
        "main": "dist/index.js",
        "types": "dist/index.d.ts",
        "dependencies": {
          "@test/core": "workspace:*"
        }
      };
      fs.writeFileSync(path.join(packageBDir, 'package.json'), JSON.stringify(packageBJson, null, 2));

      const uiCode = `
import { BaseService, Config, createConfig } from '@test/core';

export interface ComponentProps {
  title: string;
  visible: boolean;
}

export class UIService extends BaseService {
  constructor() {
    const config = createConfig('https://ui-api.example.com');
    super(config);
  }
  
  async renderComponent(props: ComponentProps): Promise<string> {
    const data = await this.makeRequest('/components');
    return \`<div>\${props.title}</div>\`;
  }
}

export function createButton(props: ComponentProps): string {
  return \`<button>\${props.title}</button>\`;
}
`;
      fs.writeFileSync(path.join(packageBDir, 'src', 'index.ts'), uiCode);

      // Create root tsconfig
      const rootTsConfig = {
        "compilerOptions": {
          "target": "ES2020",
          "module": "ESNext",
          "moduleResolution": "node",
          "strict": true,
          "esModuleInterop": true,
          "skipLibCheck": true,
          "forceConsistentCasingInFileNames": true,
          "baseUrl": ".",
          "paths": {
            "@test/*": ["packages/*/src"]
          }
        },
        "references": [
          { "path": "./packages/core" },
          { "path": "./packages/ui" }
        ]
      };
      fs.writeFileSync(path.join(testProject.rootDir, 'tsconfig.json'), JSON.stringify(rootTsConfig, null, 2));

      // Create package tsconfigs
      const packageTsConfig = {
        "compilerOptions": {
          "target": "ES2020",
          "module": "ESNext",
          "moduleResolution": "node",
          "strict": true,
          "esModuleInterop": true,
          "skipLibCheck": true,
          "forceConsistentCasingInFileNames": true,
          "declaration": true,
          "outDir": "dist",
          "baseUrl": ".",
          "paths": {
            "@test/*": ["../*/src"]
          }
        },
        "include": ["src/**/*"],
        "exclude": ["node_modules", "dist"]
      };
      
      fs.writeFileSync(path.join(packageADir, 'tsconfig.json'), JSON.stringify(packageTsConfig, null, 2));
      fs.writeFileSync(path.join(packageBDir, 'tsconfig.json'), JSON.stringify(packageTsConfig, null, 2));

      // Test MonorepoUtils integration
      const isMonorepo = MonorepoUtils.isMonorepo(testProject.rootDir);
      expect(isMonorepo).toBe(true);

      const packages = MonorepoUtils.getMonorepoPackages(testProject.rootDir);
      expect(packages).toHaveLength(2);
      expect(packages.map(p => p.name)).toContain('@test/core');
      expect(packages.map(p => p.name)).toContain('@test/ui');

      // Test RepositoryParser integration with all modules
      const parser = new RepositoryParser(testProject.rootDir);
      const result = await parser.parseRepository(testProject.rootDir);

      // Verify modules are parsed correctly
      expect(Object.keys(result.Modules)).toContain('@test/core');
      expect(Object.keys(result.Modules)).toContain('@test/ui');

      // Verify core module structure
      const coreModule = result.Modules['@test/core'];
      expect(coreModule).toBeDefined();
      expect(coreModule.Packages.src).toBeDefined();
      expect(coreModule.Packages.src.Types['Config']).toBeDefined();
      expect(coreModule.Packages.src.Types['BaseService']).toBeDefined();
      expect(coreModule.Packages.src.Functions['createConfig']).toBeDefined();

      // Verify UI module structure and dependencies
      const uiModule = result.Modules['@test/ui'];
      expect(uiModule).toBeDefined();
      expect(uiModule.Packages.src).toBeDefined();
      expect(uiModule.Packages.src.Types['UIService']).toBeDefined();
      expect(uiModule.Packages.src.Types['ComponentProps']).toBeDefined();
      expect(uiModule.Packages.src.Functions['createButton']).toBeDefined();

      // Verify cross-module dependencies in graph
      const graphKeys = Object.keys(result.Graph);
      expect(graphKeys.length).toBeGreaterThan(0);

      // Check that UI module has dependencies on core module
       const uiServiceKey = graphKeys.find(key => key.includes('UIService'));
       expect(uiServiceKey).toBeDefined();
       
       if (uiServiceKey) {
         const uiServiceNode = result.Graph[uiServiceKey];
         expect(uiServiceNode).toBeDefined();
         expect(uiServiceNode.Dependencies).toBeDefined();
         expect(uiServiceNode.Dependencies!.length).toBeGreaterThan(0);
         
         // Should have dependencies to BaseService and Config from core module
         const hasBaseServiceDep = uiServiceNode.Dependencies!.some(dep => 
           dep.Name.includes('BaseService') || dep.Name.includes('Config')
         );
         expect(hasBaseServiceDep).toBe(true);
       }

       // Verify reverse relationships are built correctly
       const allNodes = Object.values(result.Graph);
       expect(allNodes.length).toBeGreaterThan(0);
       
       // Check that some nodes have references (reverse relationships)
       const hasReferences = allNodes.some(node => 
         node.References && node.References.length > 0
       );
       expect(hasReferences).toBe(true);
       
       testProject.cleanup();
    });

    it('should handle complex dependency graphs with multiple inheritance levels', async () => {
       // Create a more complex structure with multiple inheritance levels
       const testProject = createPnpmWorkspaceProject([
         {
           path: 'packages/base',
           packageJson: {
             "name": "@complex/base",
             "version": "1.0.0"
           }
         },
         {
           path: 'packages/domain',
           packageJson: {
             "name": "@complex/domain",
             "version": "1.0.0",
             "dependencies": {
               "@complex/base": "1.0.0"
             }
           }
         },
         {
           path: 'packages/service',
           packageJson: {
             "name": "@complex/service",
             "version": "1.0.0",
             "dependencies": {
               "@complex/base": "1.0.0",
               "@complex/domain": "1.0.0"
             }
           }
         }
       ]);

       // Create root package.json
       const rootPackageJson = {
         "name": "complex-monorepo",
         "private": true
       };
       fs.writeFileSync(path.join(testProject.rootDir, 'package.json'), JSON.stringify(rootPackageJson, null, 2));

      const packagesDir = path.join(testProject.rootDir, 'packages');
      fs.mkdirSync(packagesDir, { recursive: true });

      // Base package
      const baseDir = path.join(packagesDir, 'base');
      fs.mkdirSync(path.join(baseDir, 'src'), { recursive: true });
      
      const basePackageJson = {
        "name": "@complex/base",
        "version": "1.0.0"
      };
      fs.writeFileSync(path.join(baseDir, 'package.json'), JSON.stringify(basePackageJson, null, 2));

      const baseCode = `
export abstract class Entity {
  abstract getId(): string;
}

export interface Repository<T> {
  save(entity: T): Promise<void>;
  findById(id: string): Promise<T | null>;
}
`;
      fs.writeFileSync(path.join(baseDir, 'src', 'index.ts'), baseCode);

       // Create tsconfig for base package
       const baseTsConfig = {
         "compilerOptions": {
           "target": "ES2020",
           "module": "ESNext",
           "moduleResolution": "node",
           "strict": true,
           "esModuleInterop": true,
           "skipLibCheck": true,
           "forceConsistentCasingInFileNames": true,
           "declaration": true,
           "outDir": "dist",
           "baseUrl": ".",
           "paths": {
             "@complex/*": ["../*/src"]
           }
         },
         "include": ["src/**/*"],
         "exclude": ["node_modules", "dist"]
       };
       fs.writeFileSync(path.join(baseDir, 'tsconfig.json'), JSON.stringify(baseTsConfig, null, 2));

       // Domain package
      const domainDir = path.join(packagesDir, 'domain');
      fs.mkdirSync(path.join(domainDir, 'src'), { recursive: true });
      
      const domainPackageJson = {
        "name": "@complex/domain",
        "version": "1.0.0",
        "dependencies": {
          "@complex/base": "workspace:*"
        }
      };
      fs.writeFileSync(path.join(domainDir, 'package.json'), JSON.stringify(domainPackageJson, null, 2));

      const domainCode = `
import { Entity, Repository } from '@complex/base/src';

export class User extends Entity {
  constructor(private name: string, private email: string) {
    super();
  }
  
  getId(): string {
    return this.email;
  }
}

export interface UserRepository extends Repository<User> {
  findByEmail(email: string): Promise<User | null>;
}
`;
      fs.writeFileSync(path.join(domainDir, 'src', 'index.ts'), domainCode);

       // Create tsconfig for domain package
       const domainTsConfig = {
         "compilerOptions": {
           "target": "ES2020",
           "module": "ESNext",
           "moduleResolution": "node",
           "strict": true,
           "esModuleInterop": true,
           "skipLibCheck": true,
           "forceConsistentCasingInFileNames": true,
           "declaration": true,
           "outDir": "dist",
           "baseUrl": ".",
           "paths": {
             "@complex/*": ["../*/src"]
           }
         },
         "include": ["src/**/*"],
         "exclude": ["node_modules", "dist"]
       };
       fs.writeFileSync(path.join(domainDir, 'tsconfig.json'), JSON.stringify(domainTsConfig, null, 2));

       // Service package
      const serviceDir = path.join(packagesDir, 'service');
      fs.mkdirSync(path.join(serviceDir, 'src'), { recursive: true });
      
      const servicePackageJson = {
        "name": "@complex/service",
        "version": "1.0.0",
        "dependencies": {
          "@complex/base": "workspace:*",
          "@complex/domain": "workspace:*"
        }
      };
      fs.writeFileSync(path.join(serviceDir, 'package.json'), JSON.stringify(servicePackageJson, null, 2));

      const serviceCode = `
import { User, Repository } from '@complex/domain/src';

export class DatabaseUserRepository extends Repository<User> {
  async save(user: User): Promise<User> {
    // Database save logic
    return user;
  }

  async findById(id: string): Promise<User | null> {
    // Database find logic
    return null;
  }

  async findByEmail(email: string): Promise<User | null> {
    // Database find by email logic
    return null;
  }
}

export class UserService {
  constructor(private userRepository: DatabaseUserRepository) {}

  async createUser(email: string, name: string): Promise<User> {
    const user = new User();
    user.email = email;
    user.name = name;
    return this.userRepository.save(user);
  }
}
`;
      fs.writeFileSync(path.join(serviceDir, 'src', 'index.ts'), serviceCode);

       // Create tsconfig for service package
       const serviceTsConfig = {
         "compilerOptions": {
           "target": "ES2020",
           "module": "ESNext",
           "moduleResolution": "node",
           "strict": true,
           "esModuleInterop": true,
           "skipLibCheck": true,
           "forceConsistentCasingInFileNames": true,
           "declaration": true,
           "outDir": "dist",
           "baseUrl": ".",
           "paths": {
             "@complex/*": ["../*/src"]
           }
         },
         "include": ["src/**/*"],
         "exclude": ["node_modules", "dist"]
       };
       fs.writeFileSync(path.join(serviceDir, 'tsconfig.json'), JSON.stringify(serviceTsConfig, null, 2));

       // Parse the repository
      const parser = new RepositoryParser(testProject.rootDir);
      const result = await parser.parseRepository(testProject.rootDir);

      // Verify all modules are detected
      expect(Object.keys(result.Modules)).toContain('@complex/base');
      expect(Object.keys(result.Modules)).toContain('@complex/domain');
      expect(Object.keys(result.Modules)).toContain('@complex/service');

      // Verify inheritance chains are captured
      const domainModule = result.Modules['@complex/domain'];
      const userType = domainModule.Packages.src.Types['User'];
      expect(userType).toBeDefined();
      expect(userType.Exported).toBe(true);

      const serviceModule = result.Modules['@complex/service'];
      const userServiceType = serviceModule.Packages.src.Types['UserService'];
      expect(userServiceType).toBeDefined();

      // Verify complex dependency graph
      const graphKeys = Object.keys(result.Graph);
      expect(graphKeys.length).toBeGreaterThan(0);

      // Verify complex dependencies
       const serviceNodes = Object.values(result.Graph).filter(node => 
         node.Name.includes('Service') || node.Name.includes('service')
       );
       expect(serviceNodes.length).toBeGreaterThan(0);

       const serviceNode = serviceNodes[0];
       const serviceDependencies = serviceNode.Dependencies || [];
       
       // Check if service depends on domain types (User, Repository)
       const hasDomainDep = serviceDependencies.some(dep => 
         dep.Name.includes('User') || dep.Name.includes('Repository')
       );
       
       // Check if we have all three modules parsed
       const hasBaseModule = Object.keys(result.Modules).some(key => key.includes('@complex/base'));
       const hasDomainModule = Object.keys(result.Modules).some(key => key.includes('@complex/domain'));
       const hasServiceModule = Object.keys(result.Modules).some(key => key.includes('@complex/service'));
       
       // Verify that all modules are detected and service has domain dependencies
       expect(hasBaseModule).toBe(true);
       expect(hasDomainModule).toBe(true);
       expect(hasServiceModule).toBe(true);
       expect(hasDomainDep).toBe(true);
       
       testProject.cleanup();
    });
  });

  // Tests for parseRepository method (lines 47-67)
  describe('parseRepository - Core Logic (Lines 47-67)', () => {
    describe('Monorepo Detection and Mode Handling', () => {
      it('should handle monorepo with separate mode', async () => {
        // Create Eden monorepo with multiple packages
        const testProject = createEdenMonorepoProject([
          { 
            path: 'packages/core', 
            shouldPublish: true,
            packageJson: {
              name: '@test/core',
              version: '1.0.0',
              main: 'dist/index.js'
            }
          },
          { 
            path: 'packages/utils', 
            shouldPublish: false,
            packageJson: {
              name: '@test/utils',
              version: '1.0.0',
              main: 'dist/index.js'
            }
          }
        ]);

        // Create some TypeScript files in packages
        const coreDir = path.join(testProject.rootDir, 'packages/core/src');
        fs.mkdirSync(coreDir, { recursive: true });
        fs.writeFileSync(path.join(coreDir, 'index.ts'), `
export class CoreService {
  getName(): string {
    return 'core';
  }
}
        `);

        const utilsDir = path.join(testProject.rootDir, 'packages/utils/src');
        fs.mkdirSync(utilsDir, { recursive: true });
        fs.writeFileSync(path.join(utilsDir, 'index.ts'), `
export function formatString(str: string): string {
  return str.toUpperCase();
}
        `);

        // Mock the parseMonorepoSeparateMode method
        const parser = new RepositoryParser(testProject.rootDir);
        const parseMonorepoSeparateModeSpy = jest.spyOn(parser as any, 'parseMonorepoSeparateMode')
          .mockImplementation(async () => {});
        const buildGlobalGraphSpy = jest.spyOn(parser as any, 'buildGlobalGraph')
          .mockImplementation(() => {});

        // Test with separate mode
        const result = await parser.parseRepository(testProject.rootDir, { 
          monorepoMode: 'separate' 
        });

        // Verify separate mode was called
        expect(parseMonorepoSeparateModeSpy).toHaveBeenCalledWith(
          expect.any(Array),
          expect.any(Object),
          expect.objectContaining({ monorepoMode: 'separate' })
        );
        expect(buildGlobalGraphSpy).toHaveBeenCalled();
        expect(result).toBeDefined();
        expect(result.id).toBeDefined();

        parseMonorepoSeparateModeSpy.mockRestore();
        buildGlobalGraphSpy.mockRestore();
        testProject.cleanup();
      });

      it('should handle monorepo with combined mode (default)', async () => {
        // Create Eden monorepo with multiple packages
        const testProject = createEdenMonorepoProject([
          { 
            path: 'packages/core', 
            shouldPublish: true,
            packageJson: {
              name: '@test/core',
              version: '1.0.0',
              main: 'dist/index.js'
            }
          }
        ]);

        // Create some TypeScript files
        const coreDir = path.join(testProject.rootDir, 'packages/core/src');
        fs.mkdirSync(coreDir, { recursive: true });
        fs.writeFileSync(path.join(coreDir, 'index.ts'), `
export class CoreService {
  getName(): string {
    return 'core';
  }
}
        `);

        // Mock the parseMonorepoCombinedMode method
        const parser = new RepositoryParser(testProject.rootDir);
        const parseMonorepoCombinedModeSpy = jest.spyOn(parser as any, 'parseMonorepoCombinedMode')
          .mockImplementation(async () => {});
        const buildGlobalGraphSpy = jest.spyOn(parser as any, 'buildGlobalGraph')
          .mockImplementation(() => {});

        // Test with combined mode (default)
        const result = await parser.parseRepository(testProject.rootDir);

        // Verify combined mode was called
        expect(parseMonorepoCombinedModeSpy).toHaveBeenCalledWith(
          expect.any(Array),
          expect.any(Object),
          expect.objectContaining({})
        );
        expect(buildGlobalGraphSpy).toHaveBeenCalled();
        expect(result).toBeDefined();

        parseMonorepoCombinedModeSpy.mockRestore();
        buildGlobalGraphSpy.mockRestore();
        testProject.cleanup();
      });

      it('should handle monorepo with explicit combined mode', async () => {
        // Create pnpm workspace monorepo
        const testProject = createPnpmWorkspaceProject([
          { 
            path: 'packages/api',
            packageJson: {
              name: '@test/api',
              version: '1.0.0',
              main: 'dist/index.js'
            }
          }
        ]);

        // Create some TypeScript files
        const apiDir = path.join(testProject.rootDir, 'packages/api/src');
        fs.mkdirSync(apiDir, { recursive: true });
        fs.writeFileSync(path.join(apiDir, 'index.ts'), `
export class ApiService {
  getEndpoint(): string {
    return '/api/v1';
  }
}
        `);

        // Mock the parseMonorepoCombinedMode method
        const parser = new RepositoryParser(testProject.rootDir);
        const parseMonorepoCombinedModeSpy = jest.spyOn(parser as any, 'parseMonorepoCombinedMode')
          .mockImplementation(async () => {});
        const buildGlobalGraphSpy = jest.spyOn(parser as any, 'buildGlobalGraph')
          .mockImplementation(() => {});

        // Test with explicit combined mode
        const result = await parser.parseRepository(testProject.rootDir, { 
          monorepoMode: 'combined' 
        });

        // Verify combined mode was called
        expect(parseMonorepoCombinedModeSpy).toHaveBeenCalledWith(
          expect.any(Array),
          expect.any(Object),
          expect.objectContaining({ monorepoMode: 'combined' })
        );
        expect(buildGlobalGraphSpy).toHaveBeenCalled();
        expect(result).toBeDefined();

        parseMonorepoCombinedModeSpy.mockRestore();
        buildGlobalGraphSpy.mockRestore();
        testProject.cleanup();
      });

      it('should handle single project (non-monorepo)', async () => {
        // Create a simple project structure (not a monorepo)
        const testProject = createTestProject(`
export class SingleService {
  getName(): string {
    return 'single';
  }
}
        `, 'index.ts');

        // Mock console.log to verify the log message
        const consoleLogSpy = jest.spyOn(console, 'log').mockImplementation(() => {});
        
        // Mock ModuleParser and its parseModule method
        const mockModule = {
          Name: 'test-module',
          Packages: {},
          Graph: {}
        };
        
        const parseModuleSpy = jest.spyOn(ModuleParser.prototype, 'parseModule')
          .mockResolvedValue(mockModule as any);
        const buildGlobalGraphSpy = jest.spyOn(GraphBuilder, 'buildGraph')
          .mockImplementation(() => {});

        const parser = new RepositoryParser(path.dirname(testProject.sourceFile.getFilePath()));
        const result = await parser.parseRepository(path.dirname(testProject.sourceFile.getFilePath()));

        // Verify single project handling
        expect(consoleLogSpy).toHaveBeenCalledWith('Single project detected.');
        expect(parseModuleSpy).toHaveBeenCalled();
        expect(buildGlobalGraphSpy).toHaveBeenCalled();
        expect(result.Modules[mockModule.Name]).toBe(mockModule);

        consoleLogSpy.mockRestore();
        parseModuleSpy.mockRestore();
        buildGlobalGraphSpy.mockRestore();
        testProject.cleanup();
      });

      it('should default to combined mode when monorepoMode is not specified', async () => {
        // Create Eden monorepo
        const testProject = createEdenMonorepoProject([
          { 
            path: 'packages/default-test',
            packageJson: {
              name: '@test/default-test',
              version: '1.0.0'
            }
          }
        ]);

        // Create TypeScript file
        const defaultDir = path.join(testProject.rootDir, 'packages/default-test/src');
        fs.mkdirSync(defaultDir, { recursive: true });
        fs.writeFileSync(path.join(defaultDir, 'index.ts'), `
export const DEFAULT_VALUE = 'test';
        `);

        // Mock the parseMonorepoCombinedMode method
        const parser = new RepositoryParser(testProject.rootDir);
        const parseMonorepoCombinedModeSpy = jest.spyOn(parser as any, 'parseMonorepoCombinedMode')
          .mockImplementation(async () => {});
        const buildGlobalGraphSpy = jest.spyOn(parser as any, 'buildGlobalGraph')
          .mockImplementation(() => {});

        // Test without specifying monorepoMode (should default to combined)
        const result = await parser.parseRepository(testProject.rootDir, {});

        // Verify combined mode was called (default behavior)
        expect(parseMonorepoCombinedModeSpy).toHaveBeenCalled();
        expect(buildGlobalGraphSpy).toHaveBeenCalled();

        parseMonorepoCombinedModeSpy.mockRestore();
        buildGlobalGraphSpy.mockRestore();
        testProject.cleanup();
      });

      it('should call buildGlobalGraph for all modes', async () => {
        // Test that buildGlobalGraph is called in all code paths
        const testProject = createEdenMonorepoProject([
          { 
            path: 'packages/graph-test',
            packageJson: {
              name: '@test/graph-test',
              version: '1.0.0'
            }
          }
        ]);

        const parser = new RepositoryParser(testProject.rootDir);
        const buildGlobalGraphSpy = jest.spyOn(parser as any, 'buildGlobalGraph')
          .mockImplementation(() => {});

        // Mock other methods to focus on buildGlobalGraph
        const parseMonorepoSeparateModeSpy = jest.spyOn(parser as any, 'parseMonorepoSeparateMode')
          .mockImplementation(async () => {});

        // Test separate mode
        await parser.parseRepository(testProject.rootDir, { monorepoMode: 'separate' });
        expect(buildGlobalGraphSpy).toHaveBeenCalled();

        buildGlobalGraphSpy.mockClear();
        parseMonorepoSeparateModeSpy.mockRestore();

        // Mock combined mode
        const parseMonorepoCombinedModeSpy = jest.spyOn(parser as any, 'parseMonorepoCombinedMode')
          .mockImplementation(async () => {});

        // Test combined mode
        await parser.parseRepository(testProject.rootDir, { monorepoMode: 'combined' });
        expect(buildGlobalGraphSpy).toHaveBeenCalled();

        buildGlobalGraphSpy.mockRestore();
        parseMonorepoCombinedModeSpy.mockRestore();
        testProject.cleanup();
      });
    });
  });
});