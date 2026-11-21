package classify

import (
	"context"
	"cull/internal/mail"
	"strings"
)

type Verdict string

type Classifier interface {
	Classify(ctx context.Context, h mail.Header) (Verdict, error)
}

const (
	Keep        Verdict = "keep"
	Unsubscribe Verdict = "unsubscribe"
	Junk        Verdict = "junk"
	Phishing    Verdict = "phishing"
)

var junkWords = []string{
	"act now", "limited time", "winner", "congratulations", "free gift",
	"risk-free", "click here", "opt out", "sale ends", "% off",
}

var phishWords = []string{
	"verify your account", "suspended", "unusual activity",
	"confirm your password", "final notice", "account will be closed",
}

func Classify(record mail.Header) Verdict {
	subj := strings.ToLower(record.Subject)
	if countMatches(subj, phishWords) > 0 {
		return Phishing
	}
	if countMatches(subj, junkWords) > 0 {
		return Junk
	}
	if len(record.ListUnsubscribe.Mailto) > 0 || len(record.ListUnsubscribe.URLs) > 0 {
		return Unsubscribe
	}
	return Keep
}

func countMatches(s string, words []string) int {
	count := 0
	for _, word := range words {
		if strings.Contains(s, word) {
			count++
		}
	}
	return count
}
