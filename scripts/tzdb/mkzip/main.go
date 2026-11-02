// Command mkzip writes a zoneinfo.zip of the TZif files under a directory:
// stored (no compression), entries in lexical order, no timestamps, so the
// same zone files always give the same bytes. It is the format of Go's own
// lib/time/zoneinfo.zip. Used by scripts/tzdb/update.sh (Issue #193).
//
// Usage: mkzip <zoneinfo-dir> <out.zip>
package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"hash/crc32"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 3 || !strings.HasSuffix(os.Args[2], ".zip") {
		fmt.Fprintln(os.Stderr, "usage: mkzip <zoneinfo-dir> <out.zip>")
		os.Exit(2)
	}
	count, err := build(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "mkzip:", err)
		os.Exit(1)
	}
	fmt.Printf("%d zones\n", count)
}

func build(dir, out string) (int, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	seen := map[string]bool{}
	// WalkDir visits entries in lexical order, which fixes the entry order.
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		data, err := os.ReadFile(path) // follows zic's links
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(data, []byte("TZif")) {
			return fmt.Errorf("%s is not a TZif file", name)
		}
		w, err := zw.CreateRaw(&zip.FileHeader{
			Name:               name,
			Method:             zip.Store,
			CRC32:              crc32.ChecksumIEEE(data),
			CompressedSize64:   uint64(len(data)),
			UncompressedSize64: uint64(len(data)),
		})
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
		seen[name] = true
		return nil
	})
	if err != nil {
		return 0, err
	}
	if err := zw.Close(); err != nil {
		return 0, err
	}
	for _, required := range []string{"America/Winnipeg", "UTC", "US/Eastern"} {
		if !seen[required] {
			return 0, fmt.Errorf("%s missing from %s", required, dir)
		}
	}
	return len(seen), os.WriteFile(out, buf.Bytes(), 0o644)
}
