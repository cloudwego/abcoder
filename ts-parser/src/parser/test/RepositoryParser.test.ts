import path from 'path';
import * as fs from 'fs';
import { RepositoryParser } from '../RepositoryParser';

describe('RepositoryParser', () => {
  describe('Eden Monorepo Support', () => {
    let tempDir: string;
    let cleanup: () => void;

    beforeEach(() => {
      const uniqueId = Date.now() + '_' + Math.random().toString(36).substring(2, 15);
      tempDir = path.join(__dirname, 'temp', 'eden-monorepo', uniqueId);
      fs.mkdirSync(tempDir, { recursive: true });
      
      cleanup = () => {
        if (fs.existsSync(tempDir)) {
          fs.rmSync(tempDir, { recursive: true, force: true });
        }
      };
    });

    afterEach(() => {
      cleanup();
    });

    it('should detect and parse Eden monorepo configuration', async () => {
      // Create Eden monorepo structure
      const edenConfig = {
        "$schema": "https://sf-unpkg-src.bytedance.net/@ies/eden-monorepo@3.8.0/lib/monorepo.schema.json",
        "config": {
          "strictNodeModules": true,
          "infraDir": "",
          "pnpmVersion": "10.12.1",
          "edenMonoVersion": "3.8.0"
        },
        "packages": [
          {
            "path": "packages/shared-utils",
            "shouldPublish": false
          },
          {
            "path": "packages/api-server",
            "shouldPublish": false
          }
        ]
      };

      // Write Eden config
      fs.writeFileSync(
        path.join(tempDir, 'eden.monorepo.json'),
        JSON.stringify(edenConfig, null, 2)
      );

      // Create package structure
      const sharedUtilsDir = path.join(tempDir, 'packages', 'shared-utils', 'src');
      fs.mkdirSync(sharedUtilsDir, { recursive: true });

      // Create shared-utils package.json
      const sharedUtilsPackageJson = {
        "name": "@test/shared-utils",
        "version": "1.0.0",
        "main": "dist/index.js",
        "types": "dist/index.d.ts"
      };
      fs.writeFileSync(
        path.join(tempDir, 'packages', 'shared-utils', 'package.json'),
        JSON.stringify(sharedUtilsPackageJson, null, 2)
      );

      // Create shared-utils TypeScript files
      const stringUtilsCode = `
/**
 * Formats a message with a prefix
 * @param message - The message to format
 * @param prefix - Optional prefix to add
 * @returns Formatted message
 */
export function formatMessage(message: string, prefix = '🚀'): string {
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
        path.join(tempDir, 'packages', 'shared-utils', 'tsconfig.json'),
        JSON.stringify(tsConfig, null, 2)
      );

      // Create API server package
      const apiServerDir = path.join(tempDir, 'packages', 'api-server', 'src');
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
        path.join(tempDir, 'packages', 'api-server', 'package.json'),
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
        path.join(tempDir, 'packages', 'api-server', 'tsconfig.json'),
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
        path.join(tempDir, 'tsconfig.json'),
        JSON.stringify(rootTsConfig, null, 2)
      );

      // Parse the repository
      const parser = new RepositoryParser(tempDir);
      const result = await parser.parseRepository(tempDir);

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
    });

    it('should handle Eden monorepo with complex package dependencies', async () => {
      // Create a more complex Eden monorepo structure
      const edenConfig = {
        "$schema": "https://sf-unpkg-src.bytedance.net/@ies/eden-monorepo@3.8.0/lib/monorepo.schema.json",
        "config": {
          "strictNodeModules": true,
          "edenMonoVersion": "3.8.0",
          "scriptName": {
            "test": ["test"],
            "build": ["build"]
          }
        },
        "packages": [
          {
            "path": "packages/core",
            "shouldPublish": true
          },
          {
            "path": "packages/ui-components",
            "shouldPublish": true
          },
          {
            "path": "apps/web-app",
            "shouldPublish": false
          }
        ]
      };

      fs.writeFileSync(
        path.join(tempDir, 'eden.monorepo.json'),
        JSON.stringify(edenConfig, null, 2)
      );

      // Create core package
      const coreDir = path.join(tempDir, 'packages', 'core', 'src');
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
      const uiDir = path.join(tempDir, 'packages', 'ui-components', 'src');
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
      const webAppDir = path.join(tempDir, 'apps', 'web-app', 'src');
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
          path.join(tempDir, pkg.path, 'package.json'),
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
          path.join(tempDir, pkg.path, 'tsconfig.json'),
          JSON.stringify(packageTsConfig, null, 2)
        );
      });

      // Parse the repository
      const parser = new RepositoryParser(tempDir);
      const result = await parser.parseRepository(tempDir);

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
    });
  });
});