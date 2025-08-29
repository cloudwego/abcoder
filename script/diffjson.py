#!/usr/bin/env python3
import argparse
import json
import sys
from pathlib import Path
from typing import Literal

from deepdiff import DeepDiff

# Define status types for clarity
Status = Literal["OK", "BAD", "FILE_ERROR"]


def format_diff_custom(diff: DeepDiff) -> str:
    """
    Formats a DeepDiff object into a custom human-readable string.
    This provides a clear, indented view of changes.
    """
    output = []

    # Helper to format a value for printing. Pretty-prints dicts/lists.
    def format_value(value):
        if isinstance(value, (dict, list)):
            return json.dumps(value, indent=2)
        return repr(value)

    # Handle changed values
    if "values_changed" in diff:
        for path, changes in diff["values_changed"].items():
            output.append(f"Value Changed at: {path}")
            output.append(f"  - old: {format_value(changes['old_value'])}")
            output.append(f"  + new: {format_value(changes['new_value'])}")
            output.append("--------------------")

    # Handle added items to lists/sets
    if "iterable_item_added" in diff:
        for path, value in diff["iterable_item_added"].items():
            output.append(f"Item Added at: {path}")
            output.append(f"  + new: {format_value(value)}")
            output.append("--------------------")

    # Handle removed items from lists/sets
    if "iterable_item_removed" in diff:
        for path, value in diff["iterable_item_removed"].items():
            output.append(f"Item Removed at: {path}")
            output.append(f"  - old: {format_value(value)}")
            output.append("--------------------")

    # Handle added keys in dictionaries
    if "dictionary_item_added" in diff:
        for path in diff["dictionary_item_added"]:
            output.append(f"Dictionary Key Added: {path}")
            output.append("--------------------")

    # Handle removed keys in dictionaries
    if "dictionary_item_removed" in diff:
        for path in diff["dictionary_item_removed"]:
            output.append(f"Dictionary Key Removed: {path}")
            output.append("--------------------")

    # Clean up the last separator for a tidy output
    if output and output[-1] == "--------------------":
        output.pop()

    return "\n".join(output)


def compare_json_files(file1_path: Path, file2_path: Path) -> Status:
    """
    Compares the content of two JSON files without printing output.

    Returns:
        "OK" if they match, "BAD" if they don't, "FILE_ERROR" on read/parse error.
    """
    try:
        with open(file1_path, "r", encoding="utf-8") as f1:
            json1 = json.load(f1)
        with open(file2_path, "r", encoding="utf-8") as f2:
            json2 = json.load(f2)
    except (FileNotFoundError, json.JSONDecodeError):
        return "FILE_ERROR"

    diff = DeepDiff(json1, json2, ignore_order=True)

    return "BAD" if diff else "OK"


def process_directory_comparison(old_dir: Path, new_dir: Path) -> bool:
    """
    Compares JSON files across two directories and prints results in a list format.
    """
    results: dict[str, list[str]] = {"OK": [], "BAD": [], "MISS": [], "NEW": []}
    old_files = {p.name for p in old_dir.glob("*.json")}
    new_files = {p.name for p in new_dir.glob("*.json")}

    for filename in sorted(old_files.intersection(new_files)):
        status = compare_json_files(old_dir / filename, new_dir / filename)
        results["BAD" if status != "OK" else "OK"].append(filename)

    for filename in sorted(old_files - new_files):
        results["MISS"].append(filename)

    for filename in sorted(new_files - old_files):
        results["NEW"].append(filename)

    for filename in results["OK"]:
        print(f"[OK  ]  {filename}")
    for filename in results["NEW"]:
        print(f"[NEW ]  {filename}")
    for filename in results["BAD"]:
        print(f"[BAD ]  {filename}", file=sys.stderr)
    for filename in results["MISS"]:
        print(f"[MISS]  {filename}", file=sys.stderr)

    return bool(results["BAD"] or results["MISS"])


def main():
    parser = argparse.ArgumentParser(
        description="Compare two JSON files or two directories of JSON files."
    )
    parser.add_argument(
        "path1", type=Path, help="Path to the first file or 'old' directory."
    )
    parser.add_argument(
        "path2", type=Path, help="Path to the second file or 'new' directory."
    )
    args = parser.parse_args()

    path1, path2 = args.path1, args.path2

    if not path1.exists() or not path2.exists():
        print(
            f"Error: Path does not exist: {path1 if not path1.exists() else path2}",
            file=sys.stderr,
        )
        return 1

    # --- Handle Directory Comparison ---
    if path1.is_dir() and path2.is_dir():
        print(f"Comparing directories:\n- Old: {path1}\n- New: {path2}\n")
        if process_directory_comparison(path1, path2):
            print("\nComparison finished with errors.", file=sys.stderr)
            return 1
        else:
            print("\nComparison finished successfully.")
            return 0

    # --- Handle Single File Comparison ---
    elif path1.is_file() and path2.is_file():
        try:
            with open(path1, "r", encoding="utf-8") as f1:
                json1 = json.load(f1)
            with open(path2, "r", encoding="utf-8") as f2:
                json2 = json.load(f2)
        except (FileNotFoundError, json.JSONDecodeError) as e:
            print(f"Error reading or parsing file: {e}", file=sys.stderr)
            return 1

        diff = DeepDiff(json1, json2, ignore_order=True)

        if diff:
            print(
                f"Differences found between '{path1.name}' and '{path2.name}':\n",
                file=sys.stderr,
            )
            # Format the diff into a custom readable format and print to stderr
            custom_output = format_diff_custom(diff)
            print(custom_output, file=sys.stderr)
            return 1
        else:
            print(f"Files '{path1.name}' and '{path2.name}' are identical.")
            return 0

    # --- Handle Invalid Input ---
    else:
        print(
            "Error: Both arguments must be files or both must be directories.",
            file=sys.stderr,
        )
        return 1


if __name__ == "__main__":
    sys.exit(main())
