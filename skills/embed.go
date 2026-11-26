// Package skills embeds the agent skill so the konsult binary can install it.
package skills

import "embed"

//go:embed konsult
var FS embed.FS
