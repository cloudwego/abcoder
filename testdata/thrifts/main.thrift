/**
 * 这是 thrift 文件的开头，通常用来定义不同编程语言生成代码时使用的命名空间。
 */
namespace go abcoder.testdata.thrifts
namespace java abcoder.testdata.thrifts

// 你也可以在这里包含其他的 thrift 文件
// include "shared.thrift"
include "person/person.thrift"

// 定义一个常量
const i32 VERSION = 1;

/**
 * 枚举（Enum）类型，用于定义一组命名的常量。
 */
enum Status {
  OK = 0
  ERROR = 1
}

/**
 * 结构体（Struct）是 Thrift 中的基本构建块。
 * 它们本质上等同于类，但是没有继承。
 */
struct UserProfile {
  1: required           i32    uid (api.get="/hello"),
  2: required           string name (api.get="/hello"),
  3: optional           string email (api.get="/hello"),
  4: map<string,string> attributes,
}

// sayHello 方法的请求体
struct HelloRequest {
  1: required string       name,
  2: optional UserProfile profile (api.get="/hello"),
}

// sayHello 方法的响应体
struct HelloResponse {
  1: required string message,
  2: optional Status status = Status.OK,
  3: person.Person person
}

/**
 * 异常（Exception）在功能上等同于结构体，
 * 不同之处在于它们在目标语言中会继承原生的异常基类。
 */
exception InvalidRequest {
  1: i32    code,
  2: string reason,
}

/**
 * 服务（Service）定义了你的 RPC 公共接口。
 * 代码生成器会为你创建客户端和服务器的存根（stubs）。
 */
service Greeter {
  //  一个简单的函数，返回一句问候。
  //   它可能会抛出 InvalidRequest 异常。
  HelloResponse sayHello(1: HelloRequest request) throws (1: InvalidRequest err),

  /**
   * 'oneway' 函数表示客户端发送请求后不会等待服务器的响应。
   * 客户端不会阻塞，服务器也不会发送回包。
   * Oneway 函数的返回类型必须是 void。
   */
  oneway void ping(),
}