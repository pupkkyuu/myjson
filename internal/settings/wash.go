package settings

import "myjson/internal/palette"

// defaultWash 是没选过主题时用的那一套。
const defaultWash = palette.Default

// washIDs 是可选的主题，顺序即设置里的顺序。
var washIDs = palette.IDs()
