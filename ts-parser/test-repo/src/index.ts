import express from 'express';
import cors from 'cors';
import helmet from 'helmet';
import mongoose from 'mongoose';
import jwt from 'jsonwebtoken';
import bcrypt from 'bcrypt';
import { config } from './config/app.config';
import { UserController } from '@/controllers/UserController';
import { AuthMiddleware } from '@/middleware/AuthMiddleware';
import { DatabaseService } from '@/services/DatabaseService';
import { Logger } from '@/utils/Logger';
import { validateRequest } from '@/middleware/Validation';
import { UserRoutes } from '@/routes/UserRoutes';
import { AuthRoutes } from '@/routes/AuthRoutes';
import * as fs from 'fs';
import * as path from 'path';
import { promisify } from 'util';
import * as JSON5 from 'json5';

// Re-exports
export { UserController } from '@/controllers/UserController';
export { AuthMiddleware } from '@/middleware/AuthMiddleware';

const app = express();
const PORT = process.env.PORT || 3000;

// Middleware
app.use(helmet());
app.use(cors());
app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// Database connection
const dbService = new DatabaseService();
dbService.connect().then(() => {
  Logger.info('Database connected successfully');
}).catch((error) => {
  Logger.error('Database connection failed:', error);
});

// Routes
app.use('/api/users', UserRoutes);
app.use('/api/auth', AuthRoutes);

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'OK', timestamp: new Date().toISOString() });
});

// File operations using Node.js system modules
const readFileAsync = promisify(fs.readFile);
const writeFileAsync = promisify(fs.writeFile);

app.get('/api/config', async (req, res) => {
  try {
    const configPath = path.join(__dirname, '../config.json');
    const configData = await readFileAsync(configPath, 'utf-8');
    res.json(JSON5.parse(configData));
  } catch (error) {
    res.status(500).json({ error: 'Failed to read config' });
  }
});

// JWT token generation
app.post('/api/token', (req, res) => {
  const { userId } = req.body;
  const token = jwt.sign({ userId }, config.jwtSecret, { expiresIn: '1h' });
  res.json({ token });
});

// Password hashing
app.post('/api/hash-password', async (req, res) => {
  const { password } = req.body;
  const hashedPassword = await bcrypt.hash(password, 10);
  res.json({ hashedPassword });
});

// Error handling middleware
app.use((error: Error, req: express.Request, res: express.Response, next: express.NextFunction) => {
  Logger.error('Error:', error);
  res.status(500).json({ error: 'Internal server error' });
});

// Start server
app.listen(PORT, () => {
  Logger.info(`Server running on port ${PORT}`);
});

export default app;