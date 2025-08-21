import { Project, ts } from 'ts-morph';
import * as path from 'path';
import * as fs from 'fs';
import { Repository, Node, Relation, Identity, Function, Type, Var } from '../types/uniast';
import { ModuleParser } from './ModuleParser';
import { TsConfigCache } from '../utils/tsconfig-cache';

export class RepositoryParser {
  private project: Project;
  private moduleParser: ModuleParser;
  private tsConfigCache: TsConfigCache;

  constructor(projectRoot: string, tsConfigPath?: string) {
    this.tsConfigCache = TsConfigCache.getInstance();

    

    // 设置全局 tsconfig 路径（如果提供）
    if (tsConfigPath) {
      let absoluteTsConfigPath = tsConfigPath;
      if (!path.isAbsolute(absoluteTsConfigPath)) {
        absoluteTsConfigPath = path.join(projectRoot, absoluteTsConfigPath);
      }
      this.tsConfigCache.setGlobalConfigPath(absoluteTsConfigPath);
    }
    
    // get current tsconfig path
    const configPath = this.tsConfigCache.getCurrentConfigPath(projectRoot);
    
    if (fs.existsSync(configPath)) {
      // if tsconfig.json exists, use it to configure the project
      this.project = new Project({
        tsConfigFilePath: configPath,
        compilerOptions: {
          allowJs: true,
          skipLibCheck: true,
          forceConsistentCasingInFileNames: true
        }
      });
      const tsConfigQueue: string[] = [configPath];
      const processedTsConfigs = new Set<string>();
      while (tsConfigQueue.length > 0) {
        const currentTsConfig = path.resolve(tsConfigQueue.shift()!);
        if (processedTsConfigs.has(currentTsConfig)) {
          continue;
        }
        processedTsConfigs.add(currentTsConfig);

        const tsConfig_ = ts.readConfigFile(
          currentTsConfig, ts.sys.readFile
        );
        if(tsConfig_.error) {
          console.warn("parse tsconfig error", tsConfig_.error)
          continue;
        }
        const parsedConfig = ts.parseJsonConfigFileContent(
          tsConfig_.config,
          ts.sys,
          path.dirname(currentTsConfig)
        );
        if(parsedConfig.errors.length > 0) {
          parsedConfig.errors.forEach(err => {
            console.warn("parse tsconfig warning:", err.messageText)
          });
        }
        this.project.addSourceFilesAtPaths(parsedConfig.fileNames);
        // Get references
        const references = parsedConfig.projectReferences;
        if (!references) {
          continue;
        }
        for (const ref of references) {
          const resolvedRef = ts.resolveProjectReferencePath(ref);
          if (resolvedRef.length > 0) {
            const refPath = path.resolve(path.dirname(currentTsConfig), resolvedRef);
            if(fs.existsSync(refPath)) {
              tsConfigQueue.push(refPath);
            }
          }
        }
      }
    } else {
      // if tsconfig.json does not exist, use default configuration
      this.project = new Project({
        compilerOptions: {
          target: 99,
          module: 1,
          allowJs: true,
          checkJs: false,
          skipLibCheck: true,
          skipDefaultLibCheck: true,
          strict: false,
          noImplicitAny: false,
          strictNullChecks: false,
          strictFunctionTypes: false,
          strictBindCallApply: false,
          strictPropertyInitialization: false,
          noImplicitReturns: false,
          noFallthroughCasesInSwitch: false,
          noUncheckedIndexedAccess: false,
          noImplicitOverride: false,
          noPropertyAccessFromIndexSignature: false,
          allowUnusedLabels: false,
          allowUnreachableCode: false,
          exactOptionalPropertyTypes: false,
          noImplicitThis: false,
          alwaysStrict: false,
          noImplicitUseStrict: false,
          forceConsistentCasingInFileNames: true
        }
      });
    }
    
    this.moduleParser = new ModuleParser(this.project, projectRoot);
  }

  /**
   * get current tsconfig path
   */
  public static getTsConfigPath(): string | null {
    return TsConfigCache.getInstance().getCurrentConfigPath('');
  }

  async parseRepository(repoPath: string, options: { loadExternalSymbols?: boolean, noDist?: boolean, srcPatterns?: string[] } = {}): Promise<Repository> {
    const absolutePath = path.resolve(repoPath);
    
    const repository: Repository = {
      ASTVersion: "v0.1.3",
      id: path.basename(absolutePath),
      Modules: {},
      Graph: {}
    };

    // Parse main module only
    const mainModule = await this.moduleParser.parseModule(absolutePath, '.', options);
    repository.Modules[mainModule.Name] = mainModule;

    // Build global symbol graph
    this.buildGlobalGraph(repository);

    return repository;
  }


  private buildGlobalGraph(repository: Repository): void {
    // First pass: Create all nodes from functions, types, and variables
    for (const [moduleKey, module] of Object.entries(repository.Modules)) {
      for (const [packageKey, pkg] of Object.entries(module.Packages)) {
        // Add functions to graph
        for (const [funcName, func] of Object.entries(pkg.Functions)) {
          const nodeKey = this.createNodeKey(func.ModPath, func.PkgPath, func.Name);
          const node: Node = {
            ModPath: func.ModPath,
            PkgPath: func.PkgPath,
            Name: func.Name,
            Type: 'FUNC'
          };
          
          // Add dependencies from function
          node.Dependencies = this.extractDependenciesFromFunction(func, repository);
          node.References = this.extractReferencesFromFunction(func, repository);
          
          repository.Graph[nodeKey] = node;
        }

        // Add types to graph
        for (const [typeName, type] of Object.entries(pkg.Types)) {
          const nodeKey = this.createNodeKey(type.ModPath, type.PkgPath, type.Name);
          const node: Node = {
            ModPath: type.ModPath,
            PkgPath: type.PkgPath,
            Name: type.Name,
            Type: 'TYPE'
          };
          
          // Add implements relationships
          if (type.Implements && type.Implements.length > 0) {
            node.Implements = type.Implements.map(impl => this.createRelation(impl, 'Implement'));
          }
          
          repository.Graph[nodeKey] = node;
        }

        // Add variables to graph
        for (const [varName, variable] of Object.entries(pkg.Vars)) {
          const nodeKey = this.createNodeKey(variable.ModPath, variable.PkgPath, variable.Name);
          const node: Node = {
            ModPath: variable.ModPath,
            PkgPath: variable.PkgPath,
            Name: variable.Name,
            Type: 'VAR'
          };
          
          // Add dependencies from variable
          if (variable.Dependencies && variable.Dependencies.length > 0) {
            node.Dependencies = variable.Dependencies.map(dep => this.createRelation(dep, 'Dependency'));
          }
          
          // Add groups from variable
          if (variable.Groups && variable.Groups.length > 0) {
            node.Groups = variable.Groups.map(group => this.createRelation(group, 'Group'));
          }
          
          repository.Graph[nodeKey] = node;
        }
      }
    }

    // Second pass: Add reverse relationships (References)
    this.buildReverseRelationships(repository);
  }

  private createNodeKey(modPath: string, pkgPath: string, name: string): string {
    return `${modPath}?${pkgPath}#${name}`;
  }

  private createRelation(identity: Identity, kind: Relation['Kind']): Relation {
    return {
      ModPath: identity.ModPath,
      PkgPath: identity.PkgPath,
      Name: identity.Name,
      Kind: kind
    };
  }

  private extractDependenciesFromFunction(func: Function, repository: Repository): Relation[] {
    const dependencies: Relation[] = [];
    
    // Extract from function calls
    if (func.FunctionCalls) {
      for (const call of func.FunctionCalls) {
        dependencies.push(this.createRelation(call, 'Dependency'));
      }
    }
    
    // Extract from method calls
    if (func.MethodCalls) {
      for (const call of func.MethodCalls) {
        dependencies.push(this.createRelation(call, 'Dependency'));
      }
    }
    
    // Extract from types
    if (func.Types) {
      for (const type of func.Types) {
        dependencies.push(this.createRelation(type, 'Dependency'));
      }
    }
    
    // Extract from global variables
    if (func.GlobalVars) {
      for (const globalVar of func.GlobalVars) {
        dependencies.push(this.createRelation(globalVar, 'Dependency'));
      }
    }
    
    return dependencies;
  }

  private extractReferencesFromFunction(func: Function, repository: Repository): Relation[] {
    const references: Relation[] = [];
    
    // Extract from parameters
    if (func.Params) {
      for (const param of func.Params) {
        references.push(this.createRelation(param, 'Dependency'));
      }
    }
    
    // Extract from results
    if (func.Results) {
      for (const result of func.Results) {
        references.push(this.createRelation(result, 'Dependency'));
      }
    }
    
    return references;
  }

  private buildReverseRelationships(repository: Repository): void {
    // Build a map of all relations to create reverse references
    const relationMap = new Map<string, Map<string, Relation[]>>();
    
    // Collect all relations
    for (const [nodeKey, node] of Object.entries(repository.Graph)) {
      if (node.Dependencies) {
        for (const dep of node.Dependencies) {
          const targetKey = this.createNodeKey(dep.ModPath, dep.PkgPath, dep.Name);
          if (!relationMap.has(targetKey)) {
            relationMap.set(targetKey, new Map());
          }
          if (!relationMap.get(targetKey)!.has(nodeKey)) {
            relationMap.get(targetKey)!.set(nodeKey, []);
          }
          relationMap.get(targetKey)!.get(nodeKey)!.push(dep);
        }
      }
    }
    
    // Add reverse references
    for (const [targetKey, referringNodes] of relationMap) {
      if (repository.Graph[targetKey]) {
        const references: Relation[] = [];
        for (const [sourceKey, relations] of referringNodes) {
          for (const relation of relations) {
            const sourceNode = repository.Graph[sourceKey];
            if (sourceNode) {
              references.push({
                ModPath: sourceNode.ModPath,
                PkgPath: sourceNode.PkgPath,
                Name: sourceNode.Name,
                Kind: 'Dependency'
              });
            } else {
              // Handle missing nodes with UNKNOWN type
              references.push({
                ModPath: relation.ModPath,
                PkgPath: relation.PkgPath,
                Name: relation.Name,
                Kind: 'Dependency'
              });
            }
          }
        }
        repository.Graph[targetKey].References = references;
      } else {
        // Create missing node with UNKNOWN type
        const parts = targetKey.split(/[?#]/);
        const modPath = parts[0];
        const pkgPath = parts[1];
        const name = parts[2];
        
        const missingNode: Node = {
          ModPath: modPath,
          PkgPath: pkgPath,
          Name: name,
          Type: 'UNKNOWN'
        };
        
        // Add references to the missing node
        const references: Relation[] = [];
        for (const [sourceKey, relations] of referringNodes) {
          for (const relation of relations) {
            const sourceNode = repository.Graph[sourceKey];
            if (sourceNode) {
              references.push({
                ModPath: sourceNode.ModPath,
                PkgPath: sourceNode.PkgPath,
                Name: sourceNode.Name,
                Kind: 'Dependency'
              });
            }
          }
        }
        missingNode.References = references;
        repository.Graph[targetKey] = missingNode;
      }
    }
  }
}