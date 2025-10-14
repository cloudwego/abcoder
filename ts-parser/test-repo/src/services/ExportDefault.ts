export default class ExportDefaultClass {
  public async method() {
    // 发起一个 HTTP 请求
    const response = await fetch('https://api.example.com/data');
    const data = await response.json();
    return data;
  }
}