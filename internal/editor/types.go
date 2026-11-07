package editor

import (
	"github.com/egoist/mygo/ui"
)

type Editor interface {
	Actions
	ID() string
	Title() string
	Loading() bool
	View(c *ui.Context)
}

type ComparisonEditor interface {
	Editor
	ToggleComparison()
	PreviousDifference()
	NextDifference()
}
