# Java测试仓库

这是一个用于测试Java解析器的完整Maven多模块项目。

## 项目结构

```
test-repo/
├── pom.xml                 # 父项目POM
├── core-module/           # 核心业务模块
├── service-module/        # 服务层模块
├── web-module/            # Web层模块
├── common-module/         # 通用工具模块
└── README.md
```

## 模块依赖关系

- **common-module**: 基础工具类，被所有其他模块依赖
- **core-module**: 核心业务逻辑，依赖common-module
- **service-module**: 服务层，依赖core-module和common-module
- **web-module**: Web层，依赖service-module、core-module和common-module

## 功能特性

1. **实体类**: User实体继承BaseEntity基类
2. **服务层**: UserService提供用户管理功能
3. **Web API**: RESTful接口通过UserController暴露
4. **工具类**: StringUtils提供字符串处理功能
5. **配置**: Spring配置通过AppConfig统一管理

## 使用说明

### 构建项目
```bash
mvn clean install
```

### 运行应用
```bash
cd web-module
mvn spring-boot:run
```

### 测试API

- POST /api/users/register - 注册新用户
- GET /api/users/{id} - 获取用户信息
- GET /api/users/active - 获取所有活跃用户
- PUT /api/users/{id}/status - 更新用户状态
- DELETE /api/users/{id} - 删除用户
- POST /api/users/reset-password - 重置密码

## 代码引用关系

项目展示了以下Java解析器测试场景：

1. **继承关系**: User extends BaseEntity
2. **接口实现**: InMemoryUserRepository implements UserRepository
3. **泛型使用**: Optional<User>, List<User>
4. **枚举定义**: User.UserStatus
5. **注解使用**: @Service, @RestController, @Configuration
6. **依赖注入**: 构造函数注入和@Bean配置
7. **静态方法**: StringUtils工具类的使用
8. **包导入**: 跨模块的import语句
9. **异常处理**: try-catch块和自定义异常

这个测试仓库包含了丰富的Java语法特性，适合测试Java解析器的各种场景。