import * as mongoose from 'mongoose';
import { config } from '../config/app.config';
import { Logger } from '@utils/Logger';

export class DatabaseService {
  private connectionString: string;

  constructor() {
    this.connectionString = `mongodb://${config.database.host}:${config.database.port}/${config.database.name}`;
  }

  async connect(): Promise<void> {
    try {
      await mongoose.connect(this.connectionString, {
        authSource: 'admin',
        user: config.database.user,
        pass: config.database.password,
      });
      Logger.info('Connected to MongoDB');
    } catch (error) {
      Logger.error('MongoDB connection error:', error);
      throw error;
    }
  }

  async disconnect(): Promise<void> {
    try {
      await mongoose.disconnect();
      Logger.info('Disconnected from MongoDB');
    } catch (error) {
      Logger.error('MongoDB disconnection error:', error);
      throw error;
    }
  }

  async healthCheck(): Promise<boolean> {
    try {
      const state = mongoose.connection.readyState;
      return state === 1; // 1 = connected
    } catch (error) {
      Logger.error('Database health check failed:', error);
      return false;
    }
  }
}