package markdown

import (
	"regexp"
	"strings"
)

var (
	drawMarkerRe    = regexp.MustCompile(`!\[\[draw:[^\]]+\]\]`)
	headingRe       = regexp.MustCompile(`(?m)^#{1,6}\s+.*$`)
	imageRe         = regexp.MustCompile(`!\[([^\]]*)\]\([^\)]+\)`)
	linkRe          = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	blockquoteRe    = regexp.MustCompile(`(?m)^>\s?`)
	fencedCodeRe    = regexp.MustCompile("(?s)`{3}[^\n]*\n.*?`{3}")
	inlineCodeRe    = regexp.MustCompile("`([^`]+)`")
	hrRe            = regexp.MustCompile(`(?m)^[-*_]{3,}\s*$`)
	htmlTagRe       = regexp.MustCompile(`<[^>]+>`)
	wsCollapseRe    = regexp.MustCompile(`\s+`)
	emphasisStarRe  = regexp.MustCompile(`\*([^*\n]+)\*`)
	emphasisUnderRe = regexp.MustCompile(`\b_([^_\n]+)_\b`)
	tagOnlyLineRe = regexp.MustCompile(`^[ \t]*(?:#[a-zA-Z0-9_-]+[ \t]*)+$`)
)

// StripTrailingTags removes consecutive lines consisting entirely of hashtags
// from the end of the document.
func StripTrailingTags(body string) string {
	lines := strings.Split(body, "\n")
	end := len(lines)
	for end > 0 {
		trimmed := strings.TrimSpace(lines[end-1])
		if trimmed == "" {
			end--
			continue
		}
		if tagOnlyLineRe.MatchString(trimmed) {
			end--
			continue
		}
		break
	}
	return strings.TrimRight(strings.Join(lines[:end], "\n"), " \t\n\r")
}

func stripBoldItalic(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	for {
		next := emphasisStarRe.ReplaceAllString(s, "$1")
		if next == s {
			break
		}
		s = next
	}
	s = emphasisUnderRe.ReplaceAllString(s, "$1")
	return s
}

// GenerateExcerpt returns a plain-text excerpt from markdown-like body, capped at maxLen (including ellipsis when truncated).
func GenerateExcerpt(body string, maxLen int) string {
	s := body
	s = drawMarkerRe.ReplaceAllString(s, "")
	s = headingRe.ReplaceAllString(s, "")
	s = imageRe.ReplaceAllString(s, "")
	s = linkRe.ReplaceAllString(s, "$1")
	s = stripBoldItalic(s)
	s = blockquoteRe.ReplaceAllString(s, "")
	s = fencedCodeRe.ReplaceAllString(s, "")
	s = inlineCodeRe.ReplaceAllString(s, "$1")
	s = hrRe.ReplaceAllString(s, "")
	s = htmlTagRe.ReplaceAllString(s, "")
	s = wsCollapseRe.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	const ellipsis = "..."
	if maxLen <= len(ellipsis) {
		return s[:maxLen]
	}
	budget := maxLen - len(ellipsis)
	prefix := s[:budget]
	lastSpace := strings.LastIndex(prefix, " ")
	if lastSpace <= 0 {
		return prefix + ellipsis
	}
	out := strings.TrimSpace(prefix[:lastSpace])
	if out == "" {
		return prefix + ellipsis
	}
	return out + ellipsis
}
