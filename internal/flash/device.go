package flash

import (
	"gitlab.com/calyxos/device-flasher/internal/platformtools"
)

type Device struct {
	ID            string
	Codename      string
	DiscoveryTool platformtools.ToolName
}
