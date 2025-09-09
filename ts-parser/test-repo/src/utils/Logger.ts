import * as fs from 'fs';
import * as path from 'path';

export class Logger {
  private static logFile = path.join(__dirname, '../../logs/app.log');

  static info(message: string, ...args: any[]): void {
    const timestamp = new Date().toISOString();
    const logMessage = `[${timestamp}] INFO: ${message} ${args.length > 0 ? JSON.stringify(args) : ''}`;
    console.log(logMessage);
    this.writeToFile(logMessage);
  }

  static error(message: string, ...args: any[]): void {
    const timestamp = new Date().toISOString();
    const logMessage = `[${timestamp}] ERROR: ${message} ${args.length > 0 ? JSON.stringify(args) : ''}`;
    console.error(logMessage);
    this.writeToFile(logMessage);
  }

  static warn(message: string, ...args: any[]): void {
    const timestamp = new Date().toISOString();
    const logMessage = `[${timestamp}] WARN: ${message} ${args.length > 0 ? JSON.stringify(args) : ''}`;
    console.warn(logMessage);
    this.writeToFile(logMessage);
  }

  private static writeToFile(message: string): void {
    try {
      const logDir = path.dirname(this.logFile);
      if (!fs.existsSync(logDir)) {
        fs.mkdirSync(logDir, { recursive: true });
      }
      fs.appendFileSync(this.logFile, message + '\n');
    } catch (error) {
      console.error('Failed to write to log file:', error);
    }
  }
}