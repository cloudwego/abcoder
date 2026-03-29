"""
真实的 Python 模块测试文件
用于验证 file_structure 只提取顶层符号
"""

from typing import TypeAlias, List, Dict

# ============================================
# 顶层变量 (Top-level VAR)
# ============================================

TOP_STRING = "hello"
TOP_NUMBER = 42
TOP_LIST = [1, 2, 3]
TOP_DICT = {"key": "value"}

# ============================================
# 顶层类型别名 (Top-level TYPE)
# ============================================

TopType1: TypeAlias = str
TopType2: TypeAlias = int
TopGenericType: TypeAlias = List[int]
TopDictType: TypeAlias = Dict[str, int]

# ============================================
# 顶层函数 (Top-level FUNC)
# ============================================

def top_func_no_params():
    """顶层函数：无参数"""
    pass

def top_func_with_params(a: int, b: str) -> bool:
    """顶层函数：有参数和返回值"""
    return True

def top_func_calling_others():
    """顶层函数：调用其他函数和类"""
    result = helper_func()
    obj = TopClass()
    return result

# 辅助函数
def helper_func():
    return 42

# ============================================
# 顶层类 (Top-level CLASS)
# ============================================

class TopClass:
    """顶层类"""

    class_var = 10

    def method(self):
        pass

# ============================================
# 局部符号（在函数内）
# ============================================

def func_with_locals():
    """
    包含局部符号的函数
    这些不应该出现在 file_structure 中
    """

    # 局部变量
    local_var_1 = 1
    local_var_2: str = "local"

    # 局部类型别名
    LocalType: TypeAlias = int
    LocalGenericType: TypeAlias = List[str]

    # 局部函数
    def local_func():
        pass

    def local_func_with_params(x: int) -> int:
        return x * 2

    # 局部类
    class LocalClass:
        pass

    return local_var_1

# ============================================
# 嵌套局部符号
# ============================================

def func_with_nested_locals():
    """包含嵌套局部符号的函数"""

    def nested_func():
        # 嵌套局部变量
        nested_var = 100

        def deep_nested():
            deep_nested_var = 200
            return deep_nested_var

        return nested_var

    return nested_func()

# ============================================
# 类中的局部符号（方法内）
# ============================================

class ClassWithMethods:
    """包含方法的类"""

    def method_with_locals(self):
        """方法中的局部符号"""

        method_local_var = 1

        def method_local_func():
            return method_local_var

        class MethodLocalClass:
            pass

        return method_local_func()

# ============================================
# 复杂的顶层符号
# ============================================

class ComplexClass:
    """复杂类"""

    # 类变量
    attr1 = "class attr"
    attr2: int = 20

    # 方法
    def method1(self, x: int) -> int:
        return x + 1

    @staticmethod
    def static_method():
        return "static"

    @classmethod
    def class_method(cls):
        return "class"

# 继承
class ChildClass(ComplexClass):
    """子类"""

    def method2(self):
        return "child"

# ============================================
# 导出检查
# ============================================

__all__ = [
    'TOP_STRING',
    'TOP_NUMBER',
    'TopType1',
    'TopType2',
    'top_func_no_params',
    'top_func_with_params',
    'top_func_calling_others',
    'TopClass',
    'ComplexClass',
    'ChildClass',
]
