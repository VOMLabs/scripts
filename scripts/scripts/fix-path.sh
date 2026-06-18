#!/usr/bin/env bash

echo "Cleaning PATH environment..."

# ---- Rebuild PATH from a clean base ----
CLEAN_PATHS=(
"$HOME/.local/bin"
"$HOME/.local/share/JetBrains/Toolbox/scripts"
"$HOME/.bun/bin"
"$HOME/.cargo/bin"
"$HOME/go/bin"
"$HOME/.spicetify"
"$HOME/.sdkman/candidates/kotlin/current/bin"
"$HOME/.sdkman/candidates/java/current/bin"
"$HOME/Android/Sdk/platform-tools"
"$HOME/Android/Sdk/emulator"
"$HOME/Android/Sdk/cmdline-tools/latest/bin"
"/usr/local/bin"
"/usr/local/sbin"
"/usr/bin"
"/bin"
"/usr/lib/jvm/default/bin"
)

EXISTING_PATH="${PATH:-}"

# ---- Collect directories to keep ----
NEW_PATH=""
for p in "${CLEAN_PATHS[@]}"; do
    if [ -d "$p" ]; then
        NEW_PATH="$NEW_PATH:$p"
    fi
done

# ---- Also keep any additional existing entries that are valid and not already covered ----
while IFS=':' read -ra ENTRIES; do
    for entry in "${ENTRIES[@]}"; do
        if [ -d "$entry" ]; then
            case ":$NEW_PATH:" in
                *":$entry:"*) ;;
                *) NEW_PATH="$NEW_PATH:$entry" ;;
            esac
        fi
    done
done <<< "$EXISTING_PATH"

NEW_PATH="${NEW_PATH#:}"
export PATH="$NEW_PATH"

# ---- Remove duplicate entries ----
export PATH="$(echo "$PATH" | tr ':' '\n' | awk '!seen[$0]++' | tr '\n' ':' | sed 's/:$//')"

# ---- Remove common broken SDK paths ----
REMOVE_PATTERNS=(
    "^/opt/android-sdk"
    "^/usr/share/android-sdk"
    "^/opt/google"
)

for pattern in "${REMOVE_PATTERNS[@]}"; do
    export PATH="$(echo "$PATH" | tr ':' '\n' | grep -v "$pattern" | tr '\n' ':' | sed 's/:$//')"
done

echo "PATH cleaned for current session"
echo "----------------------------------"
echo "$PATH" | tr ':' '\n'

echo ""
echo "Done. Add this to your shell rc (e.g. ~/.bashrc) to make the change permanent."
