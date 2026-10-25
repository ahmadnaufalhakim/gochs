// Package sounds exposes the bundled sound assets to the application binary.
package sounds

import "embed"

// Files contains every bundled WAV asset.
//
//go:embed *.wav
var Files embed.FS
