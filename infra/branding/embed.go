package branding

import "embed"

// Files contains the branding assets embedded into the binary.
//
//go:embed all:*
var Files embed.FS
