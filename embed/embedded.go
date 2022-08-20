package embedded

import "embed"

//go:embed static
var Content embed.FS
