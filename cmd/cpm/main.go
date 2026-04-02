package main

import "github.com/liasica/cpm/internal/cmd"

// 通过 ldflags 注入版本号
var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
