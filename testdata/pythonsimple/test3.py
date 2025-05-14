class IntVariant:
    def __init__(self, value: int):
        self.value: int = value

    def __repr__(self) -> str:
        return f"IntVariant({self.value})"


class CharVariant:
    def __init__(self, value: int):
        if not (0 <= value <= 255):
            raise ValueError(
                "CharVariant value must be an integer between 0 and 255 (u8 equivalent)"
            )
        self.value: int = value

    def __repr__(self) -> str:
        return f"CharVariant(value={self.value}, char='{chr(self.value)}')"
