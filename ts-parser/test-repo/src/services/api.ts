import * as userModel from '../models/user';
import { IUser } from "./UserService"

export interface ApiResponse<T> {
  data: T;
  success: boolean;
  message?: string;
}

export class ApiService {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  public async getUser(id: string): Promise<ApiResponse<IUser>> {
    var u = new userModel.User('Bob', 30);
    // Mock API call
    return {
      data: u,
      success: true
    };
  }

  public static createService(url: string): ApiService {
    return new ApiService(url);
  }
}


export default () => {
  return ApiService.createService('https://api.example.com');
}