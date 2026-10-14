package sidebar

type Tab struct {
	ID    string
	Title string
}

type Options struct {
	Tabs     []Tab
	ActiveID string
	TopInset float32
	Disabled bool
}

type Action uint8

const (
	NoAction Action = iota
	NewFormat
	NewComparison
	ActivateTab
	CloseTab
	ToggleSidebar
)

type Event struct {
	Action Action
	TabID  string
}
