package toolbar

import (
	"myjson/internal/locales"
	"myjson/pkg/icons"
)

const (
	NoAction           Action = ""
	Format             Action = "format"
	Compact            Action = "compress"
	Escape             Action = "escape"
	Unescape           Action = "unescape"
	SortFields         Action = "sort"
	FoldAll            Action = "collapse"
	UnfoldAll          Action = "expand"
	ToggleWrap         Action = "wrap"
	Search             Action = "search"
	CopyAll            Action = "copy"
	Clear              Action = "clear"
	ToggleComparison   Action = "comparison-toggle"
	PreviousDifference Action = "previous-difference"
	NextDifference     Action = "next-difference"
)

func toolGroups(options Options) [][]tool {
	preview := options.Kind == DiffEditor && options.Comparing
	transformDisabled := preview
	groups := [][]tool{
		{
			{Format, icons.Braces(), locales.FormatJSON(), transformDisabled},
			{Compact, icons.Minimize2(), locales.Compact(), transformDisabled},
		},
		{
			{Escape, icons.FileCode(), locales.Escape(), transformDisabled},
			{Unescape, icons.Code(), locales.Unescape(), transformDisabled},
			{SortFields, icons.ArrowDownAZ(), locales.SortFields(), transformDisabled},
		},
		{
			{FoldAll, icons.ChevronsDownUp(), locales.FoldAll(), false},
			{UnfoldAll, icons.ChevronsUpDown(), locales.UnfoldAll(), false},
			{ToggleWrap, icons.TextWrap(), locales.SoftWrap(), false},
		},
		{
			{Search, icons.Search(), locales.Search(), false},
			{CopyAll, icons.Copy(), locales.Copy(), !options.CopyEnabled || options.Kind == DiffEditor},
			{Clear, icons.Trash(), locales.Clear(), false},
		},
	}
	if options.Kind == DiffEditor {
		label, icon := locales.CompareJSON(), icons.FileDiff()
		var comparison []tool
		if options.Comparing {
			label, icon = locales.EditJSON(), icons.Pencil()
			comparison = append(comparison,
				tool{PreviousDifference, icons.ArrowUp(), locales.PreviousDifference(), options.DifferenceCount == 0},
				tool{NextDifference, icons.ArrowDown(), locales.NextDifference(), options.DifferenceCount == 0},
			)
		}
		comparison = append(comparison, tool{ToggleComparison, icon, label, false})
		groups = append([][]tool{comparison}, groups...)
	}
	return groups
}

func isTransform(action Action) bool {
	switch action {
	case Format, Compact, Escape, Unescape, SortFields:
		return true
	default:
		return false
	}
}
