class Foo:
    def __init__(self):
        self.x = 5

    def bar(self, v: int) -> int:
        self.x += v
        return self.x


def main():
    f = Foo()
    f.bar(6)
