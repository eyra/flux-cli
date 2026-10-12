// Package skill embeds the flux agent skill, so the binary can install the
// skill that matches its own commands.
package skill

import _ "embed"

//go:embed SKILL.md
var Markdown []byte
