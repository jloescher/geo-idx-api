// Command syncspec copies docs/yaak-api-collection.json into the embed path for package openapi.
// Invoked via //go:generate from doc.go (runs with cwd internal/openapi).
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	src := filepath.Join("..", "..", "docs", "yaak-api-collection.json")
	dst := filepath.Join("openapi.json")

	in, err := os.Open(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "syncspec open src %s: %v (run from repo root: make openapi-sync)\n", src, err)
		os.Exit(1)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "syncspec create dst: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		fmt.Fprintf(os.Stderr, "syncspec copy: %v\n", err)
		os.Exit(1)
	}
}
