import * as fs from 'fs';
import * as path from 'path';
import * as JSON5 from 'json5';


export interface AppConfig {
  port: number;
  database: {
    host: string;
    port: number;
    name: string;
    user: string;
    password: string;
  };
  jwtSecret: string;
  bcryptRounds: number;
  corsOrigins: string[];
  rateLimitWindow: number;
  rateLimitMax: number;
}

const configPath = path.join(__dirname, '../../config.json');

let cachedConfig: AppConfig | null = null;

export const config: AppConfig = (() => {
  if (cachedConfig) return cachedConfig;
  
  try {
    const configData = fs.readFileSync(configPath, 'utf-8');
    cachedConfig = JSON5.parse(configData);
    if (!cachedConfig) {
      throw new Error('Config file is empty');
    }
    return cachedConfig;
  } catch (error) {
    // Fallback configuration
    cachedConfig = {
      port: 3000,
      database: {
        host: 'localhost',
        port: 27017,
        name: 'testapp',
        user: 'admin',
        password: 'password'
      },
      jwtSecret: 'your-secret-key',
      bcryptRounds: 10,
      corsOrigins: ['http://localhost:3000'],
      rateLimitWindow: 15 * 60 * 1000, // 15 minutes
      rateLimitMax: 100
    };
    return cachedConfig;
  }
})();