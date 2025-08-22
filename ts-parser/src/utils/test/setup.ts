import * as fs from 'fs';
import * as path from 'path';
import { afterEach, beforeEach, beforeAll, jest } from '@jest/globals';

// Global test setup
beforeAll(() => {
  // Clean up any existing temp directories
  const tempBaseDir = path.join(__dirname, 'temp');
  if (fs.existsSync(tempBaseDir)) {
    fs.rmSync(tempBaseDir, { recursive: true, force: true });
  }
});

beforeEach(() => {
  jest.spyOn(console, 'warn').mockImplementation(() => {});
  jest.spyOn(console, 'error').mockImplementation(() => {});
});

afterEach(() => {
  jest.restoreAllMocks();
});

// Increase timeout for complex parsing operations
jest.setTimeout(30000);