# ============================================
# 顶层符号测试（应该出现在 symbolTable 中）
# ============================================

# 顶层变量（VAR）
TOP_VAR_STRING = "hello"
TOP_VAR_NUMBER = 42
TOP_VAR_LIST = [1, 2, 3]

# 顶层类型别名（TYPE）
from typing import TypeAlias, List, Dict
TopType: TypeAlias = str
TopGenericType: TypeAlias = List[int]
TopDictType: TypeAlias = Dict[str, int]

# 顶层函数（FUNC）
def top_func_no_params():
    """顶层函数：无参数"""
    pass

def top_func_with_params(a: int, b: str) -> bool:
    """顶层函数：有参数和返回值"""
    return True

# 顶层类（CLASS）
class TopClassSimple:
    """顶层类：简单类"""
    pass

class TopClassWithMembers:
    """顶层类：包含成员"""
    # 类变量
    class_var = 10
    class_var_typed: int = 20

    # 方法
    def method_simple(self):
        pass

    def method_with_return(self) -> int:
        return 42

    @staticmethod
    def static_method():
        pass

    @classmethod
    def class_method(cls):
        pass

# ============================================
# 局部符号测试（不应该出现在 symbolTable 中）
# ============================================

def func_with_local_symbols():
    """包含局部符号的函数"""

    # 局部变量
    local_var_1 = 1
    local_var_2: str = "local"

    # 局部类型别名
    LocalType: TypeAlias = int
    LocalGenericType: TypeAlias = List[str]

    # 局部函数
    def local_func():
        pass

    def local_func_nested():
        """嵌套的局部函数"""
        # 嵌套局部变量
        nested_var = 3
        return nested_var

    # 局部类
    class LocalClass:
        local_class_var = 5

        def local_method(self):
            pass

    return local_var_1

# ============================================
# 复杂场景测试
# ============================================

class ClassWithNestedDefinitions:
    """包含嵌套定义的类"""

    def method_with_local_defs(self):
        """方法中的局部符号"""

        # 方法局部变量
        method_var = 1

        # 方法局部函数
        def method_local_func():
            return method_var

        # 方法局部类
        class MethodLocalClass:
            pass

        return method_local_func()

# ============================================
# 导入符号测试（应该出现在 import 中，但不是定义）
# ============================================

from typing import Optional, Union
import os
