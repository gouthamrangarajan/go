package act

import (
	"context"
	"cull/internal/classify"
	"cull/internal/helper"
	"cull/internal/mail"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

type Action struct {
	UID  uint32
	Kind string // "unsubscribe-oneclick" | "unsubscribe-mailto" | "move-to-junk" | "skip"
	Note string
}

func Plan(header mail.Header, verdict classify.Verdict, oneClick bool) Action {
	switch verdict {
	case classify.Unsubscribe:
		if oneClick && len(header.ListUnsubscribe.URLs) > 0 {
			return Action{UID: header.UID, Kind: "unsubscribe-oneclick", Note: header.ListUnsubscribe.URLs[0]}
		}
		if len(header.ListUnsubscribe.Mailto) > 0 {
			return Action{UID: header.UID, Kind: "unsubscribe-mailto", Note: header.ListUnsubscribe.Mailto[0]}
		}
		return Action{UID: header.UID, Kind: "skip", Note: "no safe unsubscribe path"}
	case classify.Phishing:
		return Action{UID: header.UID, Kind: "move-to-junk", Note: "phishing keywords"}
	case classify.Junk:
		return Action{UID: header.UID, Kind: "move-to-junk", Note: "junk keywords"}
	}
	return Action{UID: header.UID, Kind: "skip", Note: "keep"}
}

func OneClick(ctx context.Context, action Action) error {
	slog.Info("oneclick-send", "uid", action.UID, "url", helper.Trunc(action.Note, 10))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, action.Note, strings.NewReader("List-Unsubscribe=One-Click"))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("unsubscribe returned %s", resp.Status)
	}
	slog.Info("oneclick-response", "uid", action.UID, "status", resp.StatusCode)
	return nil
}
