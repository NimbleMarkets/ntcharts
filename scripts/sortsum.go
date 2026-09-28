//go:build ignore

// sortsum rewrites go.sum files in the order the go command writes them:
// by module path, then by semantic version, then by the "/go.mod" suffix.
// Duplicate lines are removed. It uses only the standard library so that
// scripts/tidy.sh can run it in any module, including release fixtures.
//
// Usage: go run scripts/sortsum.go FILE...
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type entry struct {
	path, version, file, hash string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run scripts/sortsum.go FILE...")
		os.Exit(2)
	}
	for _, name := range os.Args[1:] {
		if err := sortFile(name); err != nil {
			fmt.Fprintf(os.Stderr, "sortsum: %s: %v\n", name, err)
			os.Exit(1)
		}
	}
}

func sortFile(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	var entries []entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || seen[line] {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			f.Close()
			return fmt.Errorf("malformed line %q", line)
		}
		seen[line] = true
		version, file := fields[1], ""
		if i := strings.Index(version, "/"); i >= 0 {
			version, file = version[:i], version[i:]
		}
		entries = append(entries, entry{fields[0], version, file, fields[2]})
	}
	f.Close()
	if err := scanner.Err(); err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.path != b.path {
			return a.path < b.path
		}
		if a.version != b.version {
			return compareVersions(a.version, b.version) < 0
		}
		return a.file < b.file
	})
	var sb strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&sb, "%s %s%s %s\n", e.path, e.version, e.file, e.hash)
	}
	return os.WriteFile(name, []byte(sb.String()), 0o644)
}

// compareVersions orders canonical semantic versions ("vMAJOR.MINOR.PATCH",
// optionally with "-prerelease" and "+build") the way semver.Compare does:
// numerically by core version, then by prerelease identifiers, with a
// release sorting after its prereleases and build metadata ignored.
func compareVersions(v, w string) int {
	vc, vp := splitVersion(v)
	wc, wp := splitVersion(w)
	for i := 0; i < 3; i++ {
		if vc[i] != wc[i] {
			if vc[i] < wc[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case vp == wp:
		return 0
	case vp == "":
		return 1
	case wp == "":
		return -1
	}
	return comparePrerelease(vp, wp)
}

// splitVersion returns the three numeric core components and the prerelease
// string of a canonical semantic version. Malformed input sorts as zero.
func splitVersion(v string) (core [3]int, prerelease string) {
	v = strings.TrimPrefix(v, "v")
	if i := strings.Index(v, "+"); i >= 0 {
		v = v[:i]
	}
	if i := strings.Index(v, "-"); i >= 0 {
		v, prerelease = v[:i], v[i+1:]
	}
	for i, part := range strings.SplitN(v, ".", 3) {
		if i < 3 {
			core[i], _ = strconv.Atoi(part)
		}
	}
	return core, prerelease
}

func comparePrerelease(v, w string) int {
	vs, ws := strings.Split(v, "."), strings.Split(w, ".")
	for i := 0; i < len(vs) && i < len(ws); i++ {
		if vs[i] == ws[i] {
			continue
		}
		vn, verr := strconv.Atoi(vs[i])
		wn, werr := strconv.Atoi(ws[i])
		switch {
		case verr == nil && werr == nil:
			if vn < wn {
				return -1
			}
			return 1
		case verr == nil:
			return -1 // numeric identifiers sort before alphanumeric ones
		case werr == nil:
			return 1
		case vs[i] < ws[i]:
			return -1
		default:
			return 1
		}
	}
	switch {
	case len(vs) < len(ws):
		return -1
	case len(vs) > len(ws):
		return 1
	}
	return 0
}
