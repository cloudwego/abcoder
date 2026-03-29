"""
Mock Python 测试文件 - 完整版
用于验证 parser 的符号提取准确性
"""

from typing import TypeAlias, List, Optional
import os
from pathlib import Path

# ============================================
# 1. 顶层变量 (Top-level VAR)
# ============================================

TOP_STRING = "hello"
TOP_NUMBER = 42
TOP_LIST = [1, 2, 3]

# 带类型注解的变量
TYPED_VAR: int = 100

# ============================================
# 2. 顶层类型别名 (Top-level TYPE)
# ============================================

TopType1: TypeAlias = str
TopType2: TypeAlias = int
TopListType: TypeAlias = List[int]
TopOptionalType: TypeAlias = Optional[str]

# ============================================
# 3. 顶层函数 (Top-level FUNC)
# ============================================

def top_func_no_params():
    """顶层函数：无参数"""
    pass

def top_func_with_params(a: int, b: str) -> bool:
    """顶层函数：有参数和返回值"""
    return True

def top_func_calling():
    """顶层函数：调用其他函数和类"""
    result = helper_func()
    obj = SymbolA()
    return result

def helper_func():
    """辅助函数"""
    return TOP_NUMBER

def func_with_type_annotations(x: int, y: str = "default") -> List[int]:
    """带完整类型注解的函数"""
    return [x]

# ============================================
# 4. 顶层类 (Top-level CLASS)
# ============================================

class SymbolA:
    """顶层类"""
    value: str = "default"

    def get_value(self) -> str:
        return self.value


# 继承示例
class ChildClass(SymbolA):
    """子类：继承 SymbolA"""

    child_attr: int = 10

    def get_value(self) -> str:
        """方法覆盖"""
        return f"Child: {self.value}"


# 多继承示例
class Mixin:
    def mixin_method(self):
        return "mixin"


class MultiInherit(SymbolA, Mixin):
    """多继承类"""
    pass


# ============================================
# 5. 导入语句（测试导入解析）
# ============================================

# 标准导入
import sys
import os as operating_system

# from 导入
from typing import Dict, Tuple

# 带别名的导入
from pathlib import Path as FilePath


# ============================================
# 6. 装饰器（测试装饰器解析）
# ============================================

class ClassWithDecorators:
    """带装饰器的类"""

    @property
    def prop(self) -> str:
        return "property"

    @staticmethod
    def static_method():
        return "static"

    @classmethod
    def class_method(cls):
        return "class"


# ============================================
# 7. 局部符号（在函数内）
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

    # 局部函数
    def local_func():
        return local_var_1

    # 局部类
    class LocalClass:
        pass

    return local_func()


def func_with_nested():
    """包含嵌套局部符号的函数"""

    def nested_func():
        nested_var = 100
        return nested_var

    return nested_func()


# ============================================
# 8. 控制流（测试复杂表达式）
# ============================================

def func_with_control_flow(x: int) -> int:
    """包含控制流的函数"""
    if x > 0:
        return x
    else:
        return -x


def func_with_loop(items: List[int]) -> int:
    """包含循环的函数"""
    total = 0
    for item in items:
        total += item
    return total


def func_with_exception() -> str:
    """包含异常处理的函数"""
    try:
        return "success"
    except Exception as e:
        return str(e)


# ============================================
# 9. 类成员访问
# ============================================

def func_using_class_members():
    """使用类成员"""
    obj = SymbolA()
    value = obj.get_value()  # 方法调用
    attr = obj.value  # 属性访问
    return value


# ============================================
# 10. 异步函数
# ============================================

async def async_func():
    """异步函数"""
    await helper_func()
    return "async"


# ============================================
# 导出
# ============================================

__all__ = [
    'TOP_STRING',
    'TOP_NUMBER',
    'TopType1',
    'TopType2',
    'top_func_no_params',
    'top_func_calling',
    'helper_func',
    'SymbolA',
    'ChildClass',
]
