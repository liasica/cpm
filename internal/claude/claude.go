package claude

// 需要按 profile 切换的认证相关条目
var authEntries = []string{
	"Cookies",
	"Cookies-journal",
	"Local Storage",
	"Session Storage",
}

// 需要同步到独立实例的共享配置
var sharedConfigs = []string{
	"claude_desktop_config.json",
	"config.json",
}

// AuthEntries 返回需要切换的认证条目列表
func AuthEntries() []string {
	return authEntries
}

// SharedConfigs 返回需要同步到独立实例的共享配置列表
func SharedConfigs() []string {
	return sharedConfigs
}
