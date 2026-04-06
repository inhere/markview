package config


const (
	DefaultPort  = "6100"
	DefaultEntry = "README.md"
)

const (
	EnvPort  = "MKVIEW_PORT"
	EnvEntry = "MKVIEW_ENTRY"
	EnvDebug = "MKVIEW_DEBUG"
	EnvWatch = "MKVIEW_WATCH"
	// Watch directory. multi use comma split
	EnvWatchDir = "MKVIEW_WATCH_DIR"
	// Watch skip directory. multi use comma split
	//  - 前缀 override: 覆盖默认的设置, append: 追加到默认的设置
	EnvWatchSkipDir = "MKVIEW_WATCH_SKIP_DIR"
)
