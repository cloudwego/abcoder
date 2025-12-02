#!/usr/bin/env python3
import argparse
from dataclasses import dataclass
import json
import os
import re
import sys
from pathlib import Path
from typing import Any, Literal, Sequence

from deepdiff import DeepDiff

oneliner_LEN = 100

Status = Literal["OK", "BAD", "FILE_ERROR"]
path_ty = list[str | int]


@dataclass
class DiffResult:
    status: Status
    diff: DeepDiff | None
    json1: Any
    json2: Any

    color_map = {
        "new": "green",
        "removed": "red",
        "moved": "yellow",
        "info": "none",
    }

    def format(self, truncate_items: int) -> str:
        return self.format_nested(truncate_items)

    def format_flat(self, truncate_items: int) -> str:
        flat, remaining_msg = self.collect_flat_raw(truncate_items)
        output_lines = []
        for path, value in flat:
            color, msg = value[0], value[1]
            color = DiffResult.color_map[color]
            leader = {"green": "+ ", "red": "- "}.get(color, "  ")
            line = format_color(
                color, leader + f"{''.join(f'[{repr(p)}]' for p in path)}: {msg}"
            )
            output_lines.append(line)
        if remaining_msg:
            output_lines.append(remaining_msg)
        return "\n".join(output_lines)

    def collect_flat_raw(
        self, truncate_items: int
    ) -> tuple[list[tuple[path_ty, Any]], str]:
        output: list[tuple[path_ty, Any]] = []

        def add_item(accessor: str, value: Any) -> None:
            output.append((_parse_accessor(accessor), value))

        # Handle new items (dictionary_item_added and iterable_item_added)
        if self.diff is not None and "dictionary_item_added" in self.diff:
            for accessor in self.diff["dictionary_item_added"]:
                add_item(accessor, ("new", _get_accessor(self.json2, accessor)))

        if self.diff is not None and "iterable_item_added" in self.diff:
            for accessor, value in self.diff["iterable_item_added"].items():
                add_item(accessor, ("new", value))

        # Handle removed items (dictionary_item_removed and iterable_item_removed)
        if self.diff is not None and "dictionary_item_removed" in self.diff:
            for accessor, value in self.diff["dictionary_item_removed"]:
                add_item(accessor, ("removed", _get_accessor(self.json1, accessor)))

        if self.diff is not None and "iterable_item_removed" in self.diff:
            for accessor, value in self.diff["iterable_item_removed"].items():
                add_item(accessor, ("removed", value))

        # Handle changed values
        if self.diff is not None and "values_changed" in self.diff:
            for accessor, changes in self.diff["values_changed"].items():
                add_item(accessor, ("removed", changes["old_value"]))
                add_item(accessor, ("new", changes["new_value"]))

        # Handle items moved (position changes in lists)
        if (
            self.diff is not None
            and "values_changed" not in self.diff
            and "iterable_item_moved" in self.diff
        ):
            for accessor, changes in self.diff["iterable_item_moved"].items():
                add_item(accessor, ("moved", "Moved location in list."))

        # Add truncation notice if needed
        if truncate_items > 0 and len(output) > truncate_items:
            remaining_msg = f"...({len(output) - truncate_items} more items)"
            output = output[:truncate_items]
        else:
            remaining_msg = ""
        return output, remaining_msg

    @staticmethod
    def make_nested_oneliner(flat: list[tuple[path_ty, Any]]) -> dict:
        output: dict = {}
        for path, value in flat:
            color, msg = value[0], value[1]
            _set_with_ensure_strpath(
                output, [str(p) for p in path], (color, _format_value_oneliner(msg))
            )
        return output

    def format_nested(self, truncate_items: int) -> str:
        flat, remaining_msg = self.collect_flat_raw(truncate_items)
        nested = DiffResult.make_nested_oneliner(flat)
        INDENT = "    "

        def isleaf(obj: Any) -> bool:
            return isinstance(obj, list)

        def _dump_leaf(obj: list, indent: int, path: str) -> str:
            output = ""
            for index, subobj in enumerate(obj):
                color, msg = subobj[0], subobj[1]
                color = DiffResult.color_map[color]
                leader = {"green": "+ ", "red": "- "}.get(color, "  ")
                leader += INDENT * indent
                l1 = format_color(color, leader + str(msg))
                # l2 = format_color(color, leader + f"# {path=}")
                output += l1  # + "\n" + l2
                if index != len(obj) - 1:
                    output += "\n"
            return output

        def _dump(obj: dict, indent: int = 0, path: str = "") -> str:
            if isinstance(obj, list):
                return _dump_leaf(obj, indent, path)
            output = ""
            for key, value in obj.items():
                kline = "  " + INDENT * indent + f"[{key}]"
                v = value
                while not isleaf(v):
                    if len(v) > 1:
                        break
                    k, v = next(iter(v.items()))
                    kline += f"[{k}]"
                if len(v) == 1:  # parent of only one leaf, colorize it same like leaf
                    color = v[0][0]
                    kline = format_color(DiffResult.color_map[color], kline)
                output += kline + "\n"
                output += _dump(v, indent + 1, path + f"[{key}]") + "\n"
            return output.rstrip()

        return _dump(nested) + "\n" + remaining_msg


def _parse_accessor(accessor_string: str) -> path_ty:
    """
    Parses a field accessor string like "['key'][0]" into a list ['key', 0].
    This allows for programmatic access to nested JSON elements.
    """
    # Regex to find content within brackets, e.g., ['key'] or [0]
    parts = re.findall(r"\[([^\]]+)\]", accessor_string)
    keys: path_ty = []
    for part in parts:
        try:
            # Try to convert to an integer for list indices
            keys.append(int(part))
        except ValueError:
            # Otherwise, it's a string key; strip surrounding quotes
            keys.append(part.strip("'\""))
    return keys


def _delete_path(data: dict | list, path: path_ty) -> None:
    """
    Deletes a value from a nested dictionary or list based on a path.
    This function modifies the data in place. If the path is invalid
    or doesn't exist, it does nothing.
    """
    if not path:
        return

    # Traverse to the parent of the target element to delete it
    parent: Any = data
    key_to_delete = path[-1]
    path_to_parent = path[:-1]

    try:
        for key in path_to_parent:
            if isinstance(parent, dict) and isinstance(key, str):
                parent = parent[key]
            elif isinstance(parent, list) and isinstance(key, int):
                parent = parent[key]
            else:
                raise TypeError("Invalid path traversal")

        # Check if the final key/index exists in the parent before deleting
        if isinstance(parent, dict) and key_to_delete in parent:
            del parent[key_to_delete]
        elif (
            isinstance(parent, list)
            and isinstance(key_to_delete, int)
            and 0 <= key_to_delete < len(parent)
        ):
            del parent[key_to_delete]
    except (KeyError, IndexError, TypeError):
        # Path is invalid (e.g., key missing, index out of bounds). Ignore and proceed.
        pass


def _get_path(data: dict | list, path: path_ty) -> Any:
    """
    Retrieves a value from a nested dictionary or list based on a path.
    Returns None if the path is invalid or doesn't exist.
    """
    current: Any = data
    try:
        for key in path:
            if isinstance(current, dict) and isinstance(key, str):
                current = current[key]
            elif isinstance(current, list) and isinstance(key, int):
                current = current[key]
            else:
                raise TypeError("Invalid path traversal")
        return current
    except (KeyError, IndexError, TypeError):
        return None


def _get_accessor(data: dict | list, accessor_string: str) -> Any:
    if accessor_string.startswith("root"):
        accessor_string = accessor_string[4:]  # Remove 'root' prefix
    path = _parse_accessor(accessor_string)
    return _get_path(data, path)


def _set_with_ensure_strpath(data: dict, str_path: list[str], value: Any) -> bool:
    try:
        current = data
        for key in str_path[:-1]:
            current = current.setdefault(key, {})
        final_key = str_path[-1]
        if final_key not in current:
            current[final_key] = []
        current[final_key].append(value)
        return True
    except (KeyError, IndexError, TypeError):
        return False


def _format_value_oneliner(value: Any) -> str:
    res = json.dumps(value)
    if len(res) < oneliner_LEN:
        return res
    if isinstance(value, dict):
        keys_str = ", ".join(f'"{key}": ...' for key in value.keys())
        res = f"{{ {keys_str} }}"
    elif isinstance(value, list):
        res = f"[ ({len(value)} items) ]"
    if len(res) < oneliner_LEN:
        return res
    return res[:oneliner_LEN] + f"...({len(res) - oneliner_LEN} more chars)"


_color_codes = {}
_reset_code = ""


def init_colors():
    global _color_codes, _reset_code
    _color_codes = {
        "red": "\033[31m",
        "green": "\033[32m",
        "yellow": "\033[33m",
    }
    _reset_code = "\033[0m"


def format_color(color: str, text: str) -> str:
    code = _color_codes.get(color.lower())
    if code is None:
        return text
    return code + text + _reset_code


def print_color(color: str, *args, **kwargs):
    sep = kwargs.get("sep", " ")
    s = format_color(color, sep.join(str(arg) for arg in args))
    print(s, **{k: v for k, v in kwargs.items() if k not in ("sep")})


def compare_files(
    file1_path: Path, file2_path: Path, ignore_fields: list[str] | None = None
) -> DiffResult:
    """
    Compares two JSON files, optionally ignoring specified fields.

    Returns:
        A tuple containing the status ("OK", "BAD", "FILE_ERROR")
        and the DeepDiff object if differences were found.
    """
    try:
        with open(file1_path, "r", encoding="utf-8") as f1:
            json1 = json.load(f1)
        with open(file2_path, "r", encoding="utf-8") as f2:
            json2 = json.load(f2)
    except (FileNotFoundError, json.JSONDecodeError):
        return DiffResult("FILE_ERROR", None, {}, {})

    # Delete ignored fields from both JSON objects before comparison
    if ignore_fields:
        for field_accessor in ignore_fields:
            path = _parse_accessor(field_accessor)
            _delete_path(json1, path)
            _delete_path(json2, path)

    diff = DeepDiff(json1, json2, ignore_order=True)

    return (
        DiffResult("BAD", diff, json1, json2)
        if diff
        else DiffResult("OK", None, json1, json2)
    )


def compare_and_report_files(
    old_path: Path,
    new_path: Path,
    ignore_fields: list[str] | None = None,
    truncate_items: int = 100,
    verbose: bool = False,
) -> int:
    result = compare_files(old_path, new_path, ignore_fields)
    if result.status == "FILE_ERROR":
        print_color(
            "red",
            f"❌ [ERROR] reading or parsing {old_path} or {new_path}.",
            file=sys.stderr,
        )
        return 1

    if result.status == "BAD" and result.diff:
        print_color(
            "red", f"❌ [DIFF] {str(old_path):<40} <-> {new_path}", file=sys.stderr
        )
        if verbose:
            new_output = result.format(truncate_items)
            new_output = "\n[details]    ".join([""] + new_output.splitlines() + [""])
            print(new_output, file=sys.stderr)
        return 1
    else:
        print_color("green", f"✅ [IDENTICAL] {str(old_path):<40} <-> {new_path}")
        return 0


def get_compare_file_list_bothdir(
    old_dir: Path, new_dir: Path
) -> tuple[list[str], list[str], list[tuple[Path, Path]]]:
    old_files = {p.name for p in old_dir.glob("*.json")}
    new_files = {p.name for p in new_dir.glob("*.json")}
    compare_file = []
    miss_file = []
    new_file = []
    for filename in sorted(old_files.intersection(new_files)):
        compare_file.append((old_dir / filename, new_dir / filename))
    for filename in sorted(old_files - new_files):
        miss_file.append(filename)
    for filename in sorted(new_files - old_files):
        new_file.append(filename)
    return miss_file, new_file, compare_file


def get_compare_file_list(path1: Path, path2: Path) -> list[tuple[Path, Path]]:
    if not path1.exists() or not path2.exists():
        raise ValueError(
            f"Error: Path does not exist: {path1 if not path1.exists() else path2}"
        )
    if path1.is_dir() and path2.is_dir():
        miss_files, new_files, compare_files = get_compare_file_list_bothdir(
            path1, path2
        )
        for filename in miss_files:
            print_color("red", f"❌ [MISS]  {filename}", file=sys.stderr)
        for filename in new_files:
            print_color("red", f"❌ [NEW ]  {filename}")
    elif path1.is_file() and path2.is_file():
        compare_files = [(path1, path2)]
    else:
        raise ValueError(
            "Error: Both arguments must be files or both must be directories."
        )
    return compare_files


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Compare two JSON files or two directories of JSON files."
    )
    parser.add_argument(
        "path1", type=Path, help="Path to the first file or 'old' directory."
    )
    parser.add_argument(
        "path2", type=Path, help="Path to the second file or 'new' directory."
    )
    parser.add_argument(
        "-i",
        "--ignore",
        action="append",
        default=[],
        help="Field to ignore, as an accessor string. Can be used multiple times. "
        "Also reads whitespace-separated values from $DIFFJSON_IGNORE. "
        "Example: -i \"['metadata']['timestamp']\"",
    )
    parser.add_argument(
        "-t",
        "--truncate_items",
        type=int,
        default=100,
        help="Maximum number of items to output. If 0, no truncation. Default: 100",
    )
    parser.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        help="Enable verbose output for directory comparison.",
    )
    args = parser.parse_args()

    # --- Combine ignore fields from CLI and environment variable ---
    cli_ignore_fields = args.ignore
    env_ignore_str = os.environ.get("DIFFJSON_IGNORE", "")
    env_ignore_fields = env_ignore_str.split() if env_ignore_str else []
    ignore_fields = list(set(cli_ignore_fields + env_ignore_fields))

    init_colors()
    compare_files = get_compare_file_list(args.path1, args.path2)
    exit_code = 0
    for file1, file2 in compare_files:
        result = compare_and_report_files(
            file1, file2, ignore_fields, args.truncate_items, args.verbose
        )
        if result != 0:
            exit_code = result
    return exit_code


if __name__ == "__main__":
    sys.exit(main())
