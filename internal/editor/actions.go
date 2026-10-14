package editor

type Actions interface {
	Format()
	Compact()
	Escape()
	Unescape()
	SortFields()
	FoldAll()
	UnfoldAll()
	ToggleWrap()
	WrapEnabled() bool
	Search()
	CopyEnabled() bool
	CopyAll(writeClipboard func(string))
	Clear()
}
