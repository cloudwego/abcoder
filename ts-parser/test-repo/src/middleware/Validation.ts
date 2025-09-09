import { Request, Response, NextFunction } from 'express';
import { Logger } from '@utils/Logger';

export function validateRequest(schema: any) {
  return (req: Request, res: Response, next: NextFunction): void => {
    const errors: Array<{ field: string; message: string }> = [];
    
    Object.keys(schema).forEach(key => {
      const rules = schema[key];
      const value = req.body[key];
      
      if (rules.required && (value === undefined || value === null || value === '')) {
        errors.push({ field: key, message: `${key} is required` });
      }
      
      if (rules.type && value !== undefined && typeof value !== rules.type) {
        errors.push({ field: key, message: `${key} must be of type ${rules.type}` });
      }
      
      if (rules.minLength && value && value.length < rules.minLength) {
        errors.push({ field: key, message: `${key} must be at least ${rules.minLength} characters` });
      }
      
      if (rules.maxLength && value && value.length > rules.maxLength) {
        errors.push({ field: key, message: `${key} must be at most ${rules.maxLength} characters` });
      }
      
      if (rules.pattern && value && !rules.pattern.test(value)) {
        errors.push({ field: key, message: `${key} format is invalid` });
      }
    });
    
    if (errors.length > 0) {
      Logger.warn('Validation errors:', errors);
      res.status(400).json({ errors });
      return;
    }
    
    next();
  };
}