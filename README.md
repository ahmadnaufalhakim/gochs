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

## Lichess Profile

Select `Lichess` in the main menu to connect an account with a Lichess Personal
API Token. To play casual Standard games online, create a token with the
`board:play` scope (and the account/profile scope used to load your profile).
Choose `Play online` from the connected Lichess menu, select a timed preset and
color, then wait for an opponent. The Lichess Board API supports Rapid and
Classical time controls. Online games support moves by board click or
UCI input, manual board flipping, resigning, draw offers, and takeback requests.

Alternatively, provide a token for the current session without saving it:

```bash
export GOCHS_LICHESS_TOKEN='your-personal-api-token'
```

Tokens entered in Gochs are stored in `$XDG_CONFIG_HOME/gochs/lichess.json`
(or `~/.config/gochs/lichess.json`) with owner-only permissions. Use
`Disconnect` from the Lichess profile screen to delete the saved token.

## Offline Puzzle Training

Initialize the local puzzle corpus and training database before playing:

```bash
make setup-puzzles
```

This imports the bundled Lichess puzzle CSV into
`$XDG_DATA_HOME/gochs/puzzles.db` (or `~/.local/share/gochs/puzzles.db`) and
creates `$XDG_DATA_HOME/gochs/training.db`. Re-running the setup rebuilds the
puzzle corpus without deleting your offline training progress.

After connecting Lichess, select `Sync puzzle rating` and confirm to set the
initial, or rebased, offline puzzle rating from your Lichess Puzzle rating.
Thereafter, clean solves gain rating and any incorrect move, hint, or solution
reveal records a failed rated attempt. Each puzzle affects an offline profile
only once.
