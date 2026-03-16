#!/bin/bash
set -e

cd "$(dirname "$0")"

echo "Building ev..."
make build

DEST="/usr/local/bin/ev"

if [ -w "/usr/local/bin" ]; then
  cp ev "$DEST"
else
  sudo cp ev "$DEST"
fi

chmod +x "$DEST"
echo "Installed to $DEST"
echo "Run: ev --version"
