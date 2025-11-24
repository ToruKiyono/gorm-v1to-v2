package main

import (
	"flag"
	"fmt"
	"os"

	"gorm-v1to-v2/internal/transform"
)

func main() {
	path := flag.String("path", ".", "Root directory containing Go files")
	flag.Parse()

	trans := transform.Transformer{}
	if err := trans.Transform(*path); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		os.Exit(1)
	}
}
