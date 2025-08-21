import { IUser } from '@models/user';
import { User } from '@models/user';
import { Logger } from '@utils/Logger';

export { IUser } from '../models/user'; 

export class UserService {
  async createUser(userData: Partial<IUser>): Promise<IUser> {
    try {
      const user = new User(userData);
      await user.save();
      Logger.info(`User created: ${user.email}`);
      return user;
    } catch (error) {
      Logger.error('Error creating user:', error);
      throw error;
    }
  }

  async getUserById(id: string): Promise<IUser | null> {
    try {
      const user = await User.findById(id);
      return user;
    } catch (error) {
      Logger.error('Error fetching user:', error);
      throw error;
    }
  }

  async getUserByEmail(email: string): Promise<IUser | null> {
    try {
      const user = await User.findOne({ email });
      return user;
    } catch (error) {
      Logger.error('Error fetching user by email:', error);
      throw error;
    }
  }

  async updateUser(id: string, updates: Partial<IUser>): Promise<IUser | null> {
    try {
      const user = await User.findByIdAndUpdate(id, updates, { new: true });
      if (user) {
        Logger.info(`User updated: ${user.email}`);
      }
      return user;
    } catch (error) {
      Logger.error('Error updating user:', error);
      throw error;
    }
  }

  async deleteUser(id: string): Promise<void> {
    try {
      await User.findByIdAndDelete(id);
      Logger.info(`User deleted: ${id}`);
    } catch (error) {
      Logger.error('Error deleting user:', error);
      throw error;
    }
  }

  async getAllUsers(): Promise<IUser[]> {
    try {
      const users = await User.find();
      return users;
    } catch (error) {
      Logger.error('Error fetching users:', error);
      throw error;
    }
  }

  async comparePassword(): Promise<boolean> {
    return await true
  }
}