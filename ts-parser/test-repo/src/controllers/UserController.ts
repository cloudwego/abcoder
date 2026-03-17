import { Request, Response } from 'express';
import { UserService } from '@services/UserService';
import { DatabaseService } from '@services/DatabaseService';
import { Logger } from '@utils/Logger';
import { validateUser } from '@utils/Validation';
import * as bcrypt from 'bcrypt';
import * as jwt from 'jsonwebtoken';

export class UserController {
  private userService: UserService;
  private dbService: DatabaseService;

  constructor() {
    this.userService = new UserService();
    this.dbService = new DatabaseService();
  }

  async createUser(req: Request, res: Response): Promise<void> {
    try {
      const { email, password, name } = req.body;
      
      if (!validateUser({ email, password, name })) {
        res.status(400).json({ error: 'Invalid user data' });
        return;
      }

      const hashedPassword = await bcrypt.hash(password, 10);
      const user = await this.userService.createUser({
        email,
        password: hashedPassword,
        name
      });

      Logger.info(`User created: ${user.email}`);
      res.status(201).json(user);
    } catch (error) {
      Logger.error('Error creating user:', error);
      res.status(500).json({ error: 'Failed to create user' });
    }
  }

  async getUser(req: Request, res: Response): Promise<void> {
    try {
      const { id } = req.params;
      const user = await this.userService.getUserById(id);
      
      if (!user) {
        res.status(404).json({ error: 'User not found' });
        return;
      }

      res.json(user);
    } catch (error) {
      Logger.error('Error fetching user:', error);
      res.status(500).json({ error: 'Failed to fetch user' });
    }
  }

  async updateUser(req: Request, res: Response): Promise<void> {
    try {
      const { id } = req.params;
      const updates = req.body;
      
      const user = await this.userService.updateUser(id, updates);
      res.json(user);
    } catch (error) {
      Logger.error('Error updating user:', error);
      res.status(500).json({ error: 'Failed to update user' });
    }
  }

  async deleteUser(req: Request, res: Response): Promise<void> {
    try {
      const { id } = req.params;
      await this.userService.deleteUser(id);
      res.status(204).send();
    } catch (error) {
      Logger.error('Error deleting user:', error);
      res.status(500).json({ error: 'Failed to delete user' });
    }
  }

  async getAllUsers(req: Request, res: Response): Promise<void> {
    try {
      const users = await this.userService.getAllUsers();
      res.json(users);
    } catch (error) {
      Logger.error('Error fetching users:', error);
      res.status(500).json({ error: 'Failed to fetch users' });
    }
  }
}