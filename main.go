package main

import "envault/cmd"

// version is injected at build time via -ldflags "-X main.version=v1.2.3"
var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
