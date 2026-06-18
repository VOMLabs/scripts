#!/usr/bin/env python3
"""Convert plain text to Markdown via OpenRouter API."""
import json
import os
import sys
import urllib.error
import urllib.request

MODEL = "openrouter/owl-alpha"
API_URL = "https://openrouter.ai/api/v1/chat/completions"
KEY_PATH = os.path.expanduser("~/.config/scripty/or/key.secret")


def load_api_key() -> str:
    key = os.environ.get("OPENROUTER_API_KEY")
    if key:
        return key
    try:
        with open(KEY_PATH) as f:
            key = f.read().strip()
            if key:
                return key
    except FileNotFoundError:
        pass
    print(
        "Error: OpenRouter API key not found. "
        "Set OPENROUTER_API_KEY or run scripty first to configure it.",
        file=sys.stderr,
    )
    sys.exit(1)


def get_input() -> str:
    if not sys.stdin.isatty():
        return sys.stdin.read()
    if len(sys.argv) > 1:
        with open(sys.argv[1]) as f:
            return f.read()
    print(f"Usage: cat file.txt | {sys.argv[0]}  OR  {sys.argv[0]} file.txt", file=sys.stderr)
    sys.exit(1)


def main():
    api_key = load_api_key()
    text = get_input().strip()
    if not text:
        print("Error: empty input", file=sys.stderr)
        sys.exit(1)

    payload = json.dumps({
        "model": MODEL,
        "messages": [
            {
                "role": "system",
                "content": "You convert plain text into clean Markdown. "
                           "Preserve meaning, structure it well with headings, "
                           "lists, and code blocks. No commentary.",
            },
            {"role": "user", "content": text},
        ],
    }).encode()

    req = urllib.request.Request(API_URL, data=payload, method="POST")
    req.add_header("Authorization", f"Bearer {api_key}")
    req.add_header("Content-Type", "application/json")

    try:
        with urllib.request.urlopen(req) as resp:
            data = json.loads(resp.read())
            content = data["choices"][0]["message"]["content"]
            print(content)
    except urllib.error.HTTPError as e:
        print(f"API error (status {e.code}): {e.read().decode()}", file=sys.stderr)
        sys.exit(1)
    except (KeyError, IndexError, json.JSONDecodeError) as e:
        print(f"Error parsing response: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
