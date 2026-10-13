# gochs
Chess implementation in Golang

## Stockfish

Install Stockfish before selecting "Play with Stockfish":

```bash
./scripts/install-stockfish.sh
```

Set `GOCHS_STOCKFISH_PATH` when Stockfish is not on `PATH` or located at
`/usr/games/stockfish`.

Play the non-TUI version with UCI or SAN move input:

```bash
make stockfish-cli
```
