package main

import "github.com/liasica/cpm/internal/cmd"

// version is injected via ldflags at build time
var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
