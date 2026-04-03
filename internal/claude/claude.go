package claude

// Auth-related entries that are switched per profile
var authEntries = []string{
	"Cookies",
	"Cookies-journal",
	"Local Storage",
	"Session Storage",
}

// Shared configs synced to standalone instances
var sharedConfigs = []string{
	"claude_desktop_config.json",
	"config.json",
}

// AuthEntries returns the list of auth entries to switch
func AuthEntries() []string {
	return authEntries
}

// SharedConfigs returns the list of configs to sync to standalone instances
func SharedConfigs() []string {
	return sharedConfigs
}
