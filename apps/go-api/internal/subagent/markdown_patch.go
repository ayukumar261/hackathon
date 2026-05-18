package subagent

import (
	"fmt"
	"strings"
)

// PatchSection replaces the body of the first heading whose text matches
// `section` (case-insensitive, trimmed). The body runs from the line after
// the heading up to the next heading of equal-or-higher level (smaller `#`
// count) or end of doc. The heading line itself is preserved.
//
// If `section` does not exist, an error is returned and the document is
// left unchanged. The caller is expected to look up an existing heading
// via read_template rather than have a new section invented.
func PatchSection(doc, section, newBody string) (string, error) {
	section = strings.TrimSpace(section)
	if section == "" {
		return "", fmt.Errorf("section is empty")
	}
	newBody = strings.TrimRight(newBody, "\n")

	lines := strings.Split(doc, "\n")
	type heading struct {
		idx   int
		level int
		text  string
	}
	var headings []heading
	for i, ln := range lines {
		level, text, ok := parseHeading(ln)
		if !ok {
			continue
		}
		headings = append(headings, heading{idx: i, level: level, text: text})
	}

	wantLower := strings.ToLower(section)
	targetIdx := -1
	for i, h := range headings {
		if strings.ToLower(h.text) == wantLower {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		return "", fmt.Errorf("section %q not found; call read_template to see available headings", section)
	}

	target := headings[targetIdx]
	bodyEnd := len(lines)
	for j := targetIdx + 1; j < len(headings); j++ {
		if headings[j].level <= target.level {
			bodyEnd = headings[j].idx
			break
		}
	}

	var b strings.Builder
	for i := 0; i <= target.idx; i++ {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	if newBody != "" {
		b.WriteString(newBody)
		b.WriteByte('\n')
	}
	if bodyEnd < len(lines) {
		// preserve a blank line separator if the body didn't already end with one
		if newBody != "" {
			b.WriteByte('\n')
		}
		for i := bodyEnd; i < len(lines); i++ {
			b.WriteString(lines[i])
			if i < len(lines)-1 {
				b.WriteByte('\n')
			}
		}
	}
	return b.String(), nil
}

func parseHeading(line string) (level int, text string, ok bool) {
	s := line
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	if n == 0 || n > 6 {
		return 0, "", false
	}
	if n >= len(s) || s[n] != ' ' {
		return 0, "", false
	}
	return n, strings.TrimSpace(s[n+1:]), true
}
