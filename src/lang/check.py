"""检查 json 中诸符号的 StartOffset, EndOffset, Line, Content 的一致性
例如检查本项目：

    $ ./lang -d -v --no-need-comment collect go . > lang.json
    # 应当成功，尤其应当是 --zero_linebase（行号从 0 开始）
    $ python3 check.py --json lang.json --base . --zero_linebase

"""

import json
import os
import argparse
from collections import defaultdict
import sys


def trim_multiline(s, max_lines=5):
    lines = s.splitlines()
    if len(lines) > max_lines:
        return "\n".join(lines[:max_lines]) + "\n..."
    return s


def safe_decode(b):
    try:
        return b.decode("utf-8")
    except UnicodeDecodeError:
        return b.decode("utf-8", errors="replace")


def verify_function_content(
    json_path,
    base_dir=".",
    bail_on_error=False,
    filter_files=None,
    filter_funcs=None,
    zero_linebase=False,
):
    with open(json_path, "r", encoding="utf-8") as f:
        data = json.load(f)

    modules = data.get("Modules", {})
    errors = defaultdict(list)

    for module_name, module in modules.items():
        packages = module.get("Packages", {})
        for package_name, package in packages.items():
            functions = package.get("Functions", {})
            for func_name, func in functions.items():
                file_name = func.get("File")
                if not file_name:
                    continue
                if filter_files and file_name not in filter_files:
                    continue
                if filter_funcs and func_name not in filter_funcs:
                    continue

                file_path = os.path.join(base_dir, file_name)
                try:
                    with open(file_path, "rb") as src:
                        content_bytes = src.read()
                except FileNotFoundError:
                    print(f"[ERROR] File not found: {file_path}")
                    errors[file_name].append(func_name)
                    if bail_on_error:
                        sys.exit(1)
                    continue

                start = func["StartOffset"]
                end = func["EndOffset"]
                expected_content = func["Content"]
                actual_bytes = content_bytes[start:end]
                actual_content = safe_decode(actual_bytes)

                # Line check
                line_number = func["Line"]
                content_str = safe_decode(content_bytes)
                file_lines = content_str.splitlines()

                try:
                    if zero_linebase:
                        actual_line_content = file_lines[line_number].strip()
                    else:
                        actual_line_content = file_lines[line_number - 1].strip()
                except IndexError:
                    actual_line_content = "<out of range>"

                expected_line_start = (
                    expected_content.splitlines()[0].strip() if expected_content else ""
                )

                offset_match = actual_content == expected_content
                line_match = actual_line_content == expected_line_start

                print(f"[{module_name}/{package_name}] Checking function: {func_name}")
                if not offset_match:
                    print("  [Mismatch] Offset content does not match.")
                    print("  Expected:\n" + trim_multiline(expected_content))
                    print("  Actual:\n" + trim_multiline(actual_content))
                if not line_match:
                    display_line_number = line_number if zero_linebase else line_number
                    print(f"  [Mismatch] Line {display_line_number} mismatch:")
                    print(f"  Expected line: {expected_line_start}")
                    print(f"  Actual line:   {actual_line_content}")
                if not offset_match or not line_match:
                    errors[file_name].append(func_name)
                    if bail_on_error:
                        sys.exit(1)
                if offset_match and line_match:
                    print("  [OK] Function content and line verified.")
                print()

    if errors:
        print("===== MISMATCH SUMMARY =====")
        for file, funcs in errors.items():
            print(f"File: {file}")
            for func in funcs:
                print(f"  - {func}")
        print("============================")
    else:
        print("✅ All functions verified successfully!")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Verify function content from JSON and source files."
    )
    parser.add_argument(
        "--json", type=str, default="input.json", help="Path to the JSON file"
    )
    parser.add_argument(
        "--base", type=str, default=".", help="Base directory for source files"
    )
    parser.add_argument(
        "--bail_on_error", action="store_true", help="Stop at first error"
    )
    parser.add_argument(
        "--filter_file",
        type=str,
        help="Comma-separated list of files to check (e.g. 'main.go,util.go')",
    )
    parser.add_argument(
        "--filter_func",
        type=str,
        help="Comma-separated list of function names to check",
    )
    parser.add_argument(
        "--zero_linebase",
        action="store_true",
        help="Line numbers in JSON are 0-based instead of 1-based",
    )

    args = parser.parse_args()
    filter_files = set(args.filter_file.split(",")) if args.filter_file else None
    filter_funcs = set(args.filter_func.split(",")) if args.filter_func else None

    verify_function_content(
        json_path=args.json,
        base_dir=args.base,
        bail_on_error=args.bail_on_error,
        filter_files=filter_files,
        filter_funcs=filter_funcs,
        zero_linebase=args.zero_linebase,
    )
