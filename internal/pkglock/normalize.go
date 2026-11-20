// Copyright 2026 Ilean Monterrubio Jr
// SPDX-License-Identifier: Apache-2.0

package pkglock

import "bytes"

func normalize(b []byte) []byte {
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))

	lines := bytes.Split(b, []byte("\n"))
	for i, line := range lines {
		lines[i] = bytes.TrimRight(line, " \t")
	}
	b = bytes.Join(lines, []byte("\n"))
	b = bytes.TrimRight(b, "\n")
	b = append(b, '\n')
	return b
}
