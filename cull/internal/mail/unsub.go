package mail

import (
	"regexp"
	"strings"
)

type Unsub struct {
	Mailto []string
	URLs   []string
}

var urisRegex = regexp.MustCompile(`<([^>]*)>`)

func ParseListUnsub(raw string) Unsub {
	var out Unsub
	add := func(value string) {
		value = strings.TrimSpace(value)
		low := strings.ToLower(value)
		switch {
		case strings.HasPrefix(low, "mailto:"):
			out.Mailto = append(out.Mailto, value)
		case strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://"):
			out.URLs = append(out.URLs, value)
		}
	}
	if matches := urisRegex.FindAllStringSubmatch(raw, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 1 {
				add(match[1])
			}
		}
	} else {
		for _, part := range strings.Split(raw, ",") {
			add(part)
		}
	}
	return out
}
