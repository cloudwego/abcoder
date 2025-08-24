import { Router } from 'express';
import * as jwt from 'jsonwebtoken';
import * as bcrypt from 'bcrypt';
import { UserService } from '@services/UserService';
import { validateRequest } from '@middleware/Validation';
import { config } from '../config/app.config';
import { Logger } from '@utils/Logger';

const router = Router();
const userService = new UserService();

// Validation schemas
const loginSchema = {
  email: { required: true, type: 'string', pattern: /^[^\s@]+@[^\s@]+\.[^\s@]+$/ },
  password: { required: true, type: 'string', minLength: 6 }
};

const registerSchema = {
  email: { required: true, type: 'string', pattern: /^[^\s@]+@[^\s@]+\.[^\s@]+$/ },
  password: { required: true, type: 'string', minLength: 6 },
  name: { required: true, type: 'string', minLength: 2, maxLength: 50 }
};

// Routes
router.post('/register', 
  validateRequest(registerSchema),
  async (req, res) => {
    try {
      const { email, password, name } = req.body;
      
      // Check if user already exists
      const existingUser = await userService.getUserByEmail(email);
      if (existingUser) {
        return res.status(400).json({ error: 'User already exists' });
      }
      
      // Create user
      const user = await userService.createUser({ email, password, name });
      
      // Generate token
      const token = jwt.sign(
        { userId: user._id, email: user.email },
        config.jwtSecret,
        { expiresIn: '24h' }
      );
      
      Logger.info(`User registered: ${email}`);
      res.status(201).json({
        message: 'User registered successfully',
        token,
        user: {
          id: user._id,
          email: user.email,
          name: user.name
        }
      });
    } catch (error) {
      Logger.error('Registration error:', error);
      res.status(500).json({ error: 'Registration failed' });
    }
  }
);

router.post('/login',
  validateRequest(loginSchema),
  async (req, res) => {
    try {
      const { email, password } = req.body;
      
      // Find user
      const user = await userService.getUserByEmail(email);
      if (!user) {
        return res.status(401).json({ error: 'Invalid credentials' });
      }
      
      // Check password
      const isValidPassword = await user.comparePassword(password);
      if (!isValidPassword) {
        return res.status(401).json({ error: 'Invalid credentials' });
      }
      
      // Generate token
      const token = jwt.sign(
        { userId: user._id, email: user.email, role: user.role },
        config.jwtSecret,
        { expiresIn: '24h' }
      );
      
      Logger.info(`User logged in: ${email}`);
      res.json({
        message: 'Login successful',
        token,
        user: {
          id: user._id,
          email: user.email,
          name: user.name,
          role: user.role
        }
      });
    } catch (error) {
      Logger.error('Login error:', error);
      res.status(500).json({ error: 'Login failed' });
    }
  }
);

router.post('/refresh-token', async (req, res) => {
  try {
    const { token } = req.body;
    
    if (!token) {
      return res.status(401).json({ error: 'No token provided' });
    }
    
    const decoded = jwt.verify(token, config.jwtSecret) as any;
    const newToken = jwt.sign(
      { userId: decoded.userId, email: decoded.email },
      config.jwtSecret,
      { expiresIn: '24h' }
    );
    
    res.json({ token: newToken });
  } catch (error) {
    Logger.error('Token refresh error:', error);
    res.status(401).json({ error: 'Invalid token' });
  }
});

export { router as AuthRoutes };