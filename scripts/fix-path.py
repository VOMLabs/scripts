#!/usr/bin/env python3
"""Rebuild PATH from a clean base of known-good directories."""
import os
import re
import sys

CLEAN_PATHS = [
    "~/.local/bin",
    "~/.local/share/JetBrains/Toolbox/scripts",
    "~/.bun/bin",
    "~/.cargo/bin",
    "~/go/bin",
    "~/.spicetify",
    "~/.sdkman/candidates/kotlin/current/bin",
    "~/.sdkman/candidates/java/current/bin",
    "~/Android/Sdk/platform-tools",
    "~/Android/Sdk/emulator",
    "~/Android/Sdk/cmdline-tools/latest/bin",
    "/usr/local/bin",
    "/usr/local/sbin",
    "/usr/bin",
    "/bin",
    "/usr/lib/jvm/default/bin",
]

REMOVE_PATTERNS = [
    re.compile(r"^/opt/android-sdk"),
    re.compile(r"^/usr/share/android-sdk"),
    re.compile(r"^/opt/google"),
]


def main():
    print("Cleaning PATH environment...")

    existing = os.environ.get("PATH", "")
    seen = set()
    new_paths = []

    for p in CLEAN_PATHS:
        full = os.path.expanduser(p)
        if os.path.isdir(full) and full not in seen:
            new_paths.append(full)
            seen.add(full)

    for entry in existing.split(":"):
        entry = entry.strip()
        if entry and os.path.isdir(entry) and entry not in seen:
            new_paths.append(entry)
            seen.add(entry)

    filtered = [p for p in new_paths if not any(pt.search(p) for pt in REMOVE_PATTERNS)]

    os.environ["PATH"] = ":".join(filtered)

    print("PATH cleaned for current session")
    print("----------------------------------")
    for p in filtered:
        print(p)

    print()
    print("Done. Add this to your shell rc (e.g. ~/.bashrc) to make the change permanent.")


if __name__ == "__main__":
    main()
