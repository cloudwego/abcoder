import { Request, Response } from 'express';
import { NextFunction } from 'express';
import * as jwt from 'jsonwebtoken';
import { config } from '../config/app.config';
import { Logger } from '@utils/Logger';

export interface AuthRequest extends Request {
  user?: any;
}

export class AuthMiddleware {
  static authenticate(req: AuthRequest, res: Response, next: NextFunction): void {
    const token = req.header('Authorization')?.replace('Bearer ', '');

    if (!token) {
      res.status(401).json({ error: 'Access denied. No token provided.' });
      return;
    }

    try {
      const decoded = jwt.verify(token, config.jwtSecret);
      req.user = decoded;
      next();
    } catch (error) {
      Logger.error('Token verification failed:', error);
      res.status(400).json({ error: 'Invalid token.' });
    }
  }

  static authorize(roles: string[] = ['user']) {
    return (req: AuthRequest, res: Response, next: NextFunction): void => {
      if (!req.user) {
        res.status(401).json({ error: 'User not authenticated' });
        return;
      }

      if (!roles.includes(req.user.role)) {
        res.status(403).json({ error: 'Insufficient permissions' });
        return;
      }

      next();
    };
  }
}