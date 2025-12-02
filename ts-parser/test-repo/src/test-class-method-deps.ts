export type UserData = {
  id: string;
  name: string;
};

export function validateUser(user: UserData): boolean {
  return user.id.length > 0;
}

export class UserService {
  private data: UserData;

  constructor(userData: UserData) {
    this.data = userData;
  }

  // 方法中应该能识别：
  // 1. 参数类型 UserData
  // 2. 返回类型 boolean  
  // 3. 函数调用 validateUser
  checkValid(user: UserData): boolean {
    return validateUser(user);
  }

  // 方法中应该能识别：
  // 1. 返回类型 UserData
  // 2. 全局变量引用 this.data
  getUserData(): UserData {
    return this.data;
  }
}
