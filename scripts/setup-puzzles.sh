#!/usr/bin/env bash
set -euo pipefail

input="data/puzzle/lichess_db_puzzle.csv"
themes="data/puzzle/themes.json"

while (($# > 0)); do
    case "${1}" in
        --input)
            input="${2:?missing value for --input}"
            shift 2
            ;;
        --themes)
            themes="${2:?missing value for --themes}"
            shift 2
            ;;
        *)
            printf 'Usage: %s [--input CSV] [--themes JSON]\n' "${0}" >&2
            exit 2
            ;;
    esac
done

go run ./cmd/gochs-init-training
go run ./cmd/gochs-import-puzzles --input "${input}" --themes "${themes}"
