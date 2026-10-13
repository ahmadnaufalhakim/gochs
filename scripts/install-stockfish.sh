#!/usr/bin/env bash

set -euo pipefail

if command -v stockfish >/dev/null 2>&1 || [[ -x /usr/games/stockfish ]]; then
    echo "Stockfish is already installed."
    exit 0
fi

if command -v apt-get >/dev/null 2>&1; then
    sudo apt-get update
    sudo apt-get install -y stockfish
    exit 0
fi

if command -v brew >/dev/null 2>&1; then
    brew install stockfish
    exit 0
fi

echo "Unsupported package manager. Install Stockfish, then set GOCHS_STOCKFISH_PATH to its executable." >&2
exit 1
