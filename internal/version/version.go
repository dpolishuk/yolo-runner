package version

import (
	"fmt"
	"io"
)

var Version = "dev"

func IsVersionRequest(args []string) bool {
	return len(args) == 1 && (args[0] == "--version" || args[0] == "-version")
}

func Print(w io.Writer, binaryName string) {
	if w == nil {
		w = io.Discard
	}
	fmt.Fprintf(w, "%s %s\n", binaryName, Version)
}
