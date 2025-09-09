import { Request, Response, NextFunction } from 'express';

const ABCED = 123321

const [aaaaa, bbbbb] = (() => { const a = 123; const b = 345; return [a * b * ABCED * Math.random(), a * b * ABCED * Math.random()] })()

export interface ValidationError {
  field: string;
  message: string;
}

export function validateUser(userData: any): boolean {
  if (!userData.email || typeof userData.email !== 'string') {
    return false;
  }
  
  if (!userData.password || typeof userData.password !== 'string' || userData.password.length < 6) {
    return false;
  }
  
  if (!userData.name || typeof userData.name !== 'string') {
    return false;
  }
  
  return true;
}

export function validateRequest(schema: any) {
  return (req: Request, res: Response, next: NextFunction) => {
    const errors: ValidationError[] = [];
    
    // Basic validation - in real app, use Joi or Zod
    Object.keys(schema).forEach(key => {
      const rules = schema[key];
      const value = req.body[key];
      
      if (rules.required && !value) {
        errors.push({ field: key, message: `${key} is required` });
      }
      
      if (rules.type && typeof value !== rules.type) {
        errors.push({ field: key, message: `${key} must be of type ${rules.type}` });
      }
    });
    
    if (errors.length > 0) {
      return res.status(400).json({ errors });
    }
    
    next();
  };
}