package org.example;

public class TestUtf16 {
    // 测试包含emoji和中文的情况：😀 中文测试
    public void testWithUnicode() {
        String emoji = "😀";  // emoji是4字节UTF-8
        String chinese = "中文";  // 中文字符是3字节UTF-8
        String mixed = "a😀中文b";  // 混合字符串
    }
    
    // 方法参数测试
    public void methodWithParams(String param1, int param2) {
        // 测试方法定义位置
    }
}