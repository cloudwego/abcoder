#!/usr/bin/env node

import { Command } from 'commander';
import fs from 'fs';
import path from 'path';
import { RepositoryParser } from './parser/RepositoryParser';
import { JsonStreamStringify } from 'json-stream-stringify';

const program = new Command();

program
  .name('abcoder-ts-parser')
  .description('TypeScript AST parser for UNIAST specification')
  .version('0.0.21');

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

      // Output the repository JSON file
      const outputPath = path.resolve(options.output);
      const jsonStream = new JsonStreamStringify(repository, undefined, options.pretty ? 2 : undefined);
      const fileStream = fs.createWriteStream(outputPath);
      jsonStream.pipe(fileStream);

      fileStream.on('finish', () => {
        console.log(`Repository has been parsed and saved to ${outputPath}`);
      });

      fileStream.on('error', (err) => {
        console.error('Error writing to file:', err);
      });

    } catch (error) {
      console.error('Error parsing repository:', error);
      process.exit(1);
    }
  });

program.parse();