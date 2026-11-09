// Command plewtus-mcp runs the Plewtus Model Context Protocol server over stdio.
package main

import (
	"os"

	"plewtus/internal/config"
	"plewtus/internal/mcp"
)

func main() {
	os.Exit(mcp.Run(config.Load()))
}
