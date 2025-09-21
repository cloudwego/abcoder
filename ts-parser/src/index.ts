#!/usr/bin/env node

import { Command } from 'commander';
import * as path from 'path';
import { RepositoryParser } from './parser/RepositoryParser';
import * as fs from 'fs';

const program = new Command();

program
  .name('abcoder-ts-parser')
  .description('TypeScript AST parser for UNIAST v0.1.3 specification')
  .version('1.0.0');

program
  .command('parse')
  .description('Parse a TypeScript repository and generate UNIAST JSON')
  .argument('<directory>', 'Directory to parse')
  .option('-o, --output <file>', 'Output file path', 'output.json')
  .option('-t, --tsconfig <file>', 'Path to tsconfig.json file (relative to project root if not absolute)')
  .option('--no-dist', 'Ignore dist folder and its contents', false)
  .option('--pretty', 'Pretty print JSON output', false)
  .option('--src <dirs>', 'Directory paths to include (comma-separated)', (value) => value.split(','))
  .option('--monorepo-mode <mode>', '"combined"(output entrie monorep repository)  "separate"(output each app)', 'combined')
  .action(async (directory, options) => {
    try {
      const repoPath = path.resolve(directory);
      
      if (!fs.existsSync(repoPath)) {
        console.error(`Error: Directory ${repoPath} does not exist`);
        process.exit(1);
      }

      console.log(`Parsing TypeScript repository: ${repoPath}`);
      
      const parser = new RepositoryParser(repoPath, options.tsconfig);
      const repository = await parser.parseRepository(repoPath, {
        loadExternalSymbols: false,
        noDist: options.noDist,
        srcPatterns: options.src,
        monorepoMode: options.monorepoMode as 'combined' | 'separate'
      });

      // In separate mode, output both individual packages and the combined repository
      if (options.monorepoMode === 'separate') {
        
        // Output the combined repository JSON file to the specified output.json location
        console.log('Writing combined repository file...');
        const combinedOutputPath = path.resolve(options.output);
        fs.writeFileSync(combinedOutputPath, JSON.stringify(repository, null, 2));
        console.log(`Combined repository written to: ${combinedOutputPath}`);
        
      } else {
        const outputPath = path.resolve(options.output);
        const jsonOutput = options.pretty 
          ? JSON.stringify(repository, null, 2)
          : JSON.stringify(repository);

        fs.writeFileSync(outputPath, jsonOutput);
        
        console.log(`Successfully parsed repository`);
        console.log(`Output written to: ${outputPath}`);
        console.log(`Total modules: ${Object.keys(repository.Modules).length}`);
        console.log(`Total symbols in graph: ${Object.keys(repository.Graph).length}`);
      }

    } catch (error) {
      console.error('Error parsing repository:', error);
      process.exit(1);
    }
  });


program.parse();