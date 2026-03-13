#!/usr/bin/env bash
set -euo pipefail

# Download pinned mermaid.js IIFE build
MERMAID_VERSION="11.4.1"
MERMAID_URL="https://cdn.jsdelivr.net/npm/mermaid@${MERMAID_VERSION}/dist/mermaid.min.js"
DEST="web/static/mermaid.min.js"

echo "Downloading mermaid.js v${MERMAID_VERSION}..."
curl -sL "${MERMAID_URL}" -o "${DEST}"

# Verify it's a valid JS file (contains mermaid reference)
if grep -q "mermaid" "${DEST}"; then
    echo "OK: mermaid.min.js downloaded ($(wc -c < "${DEST}" | tr -d ' ') bytes)"
else
    echo "ERROR: downloaded file does not appear to be mermaid.js"
    rm -f "${DEST}"
    exit 1
fi
