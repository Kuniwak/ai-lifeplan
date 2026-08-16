package lifeplancmd

import (
	"bytes"
	"fmt"
	"sort"
)

func diffTrees(first, second map[string][]byte) []string {
	names := make(map[string]struct{}, len(first)+len(second))
	for name := range first {
		names[name] = struct{}{}
	}
	for name := range second {
		names[name] = struct{}{}
	}

	var diffs []string
	for name := range names {
		a, inFirst := first[name]
		b, inSecond := second[name]
		switch {
		case !inSecond:
			diffs = append(diffs, fmt.Sprintf("%s: written by the first run only", name))
		case !inFirst:
			diffs = append(diffs, fmt.Sprintf("%s: written by the second run only", name))
		case !bytes.Equal(a, b):
			diffs = append(diffs, fmt.Sprintf("%s: %s", name, firstDifference(a, b)))
		}
	}
	sort.Strings(diffs)
	return diffs
}

func firstDifference(a, b []byte) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	at := fmt.Sprintf("byte %d (line %d)", i, 1+bytes.Count(a[:i], []byte("\n")))

	switch {
	case i == len(a):
		return fmt.Sprintf("the first run's file ends at %s; the second goes on with %q", at, b[i:])
	case i == len(b):
		return fmt.Sprintf("the second run's file ends at %s; the first goes on with %q", at, a[i:])
	}

	lineA, lineB := lineAt(a, i), lineAt(b, i)
	if lineA == lineB {
		return fmt.Sprintf("differs at %s: %q vs %q", at, a[i], b[i])
	}
	return fmt.Sprintf("differs at %s: %q vs %q", at, lineA, lineB)
}

func lineAt(s []byte, offset int) string {
	if offset > len(s) {
		offset = len(s)
	}
	start := bytes.LastIndexByte(s[:offset], '\n') + 1
	end := bytes.IndexByte(s[start:], '\n')
	if end < 0 {
		return string(s[start:])
	}
	return string(s[start : start+end])
}
