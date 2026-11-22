package main

import (
	"context"
	"cull/internal/act"
	"cull/internal/classify"
	"cull/internal/config"
	"cull/internal/mail"
	"cull/internal/state"
	"flag"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	startTime := time.Now()
	slog.Info("info", "cull starting:", startTime.Format("15:04"))

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	apply := flag.Bool("apply", false, "execute actions (default: dry-run)")
	maxActions := flag.Int("max-actions", 5, "cap executed actions per run")
	noOfItems := flag.Int("num-items", 50, "number of recent items to fetch")
	flag.Parse()

	slog.Info("info", "apply", *apply, "max_actions", *maxActions)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	st, err := state.Load(cfg.StatePath)
	if err != nil {
		log.Fatal(err)
	}
	client, err := mail.Open(cfg, &st)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	// folders, err := client.Folders()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// for _, folder := range folders {
	// 	slog.Info("folder", "name", folder)
	// }
	// client.CheckBulkAndJunk()
	data, err := client.RecentHeaders(*noOfItems)
	if err != nil {
		log.Fatal(err)
	}
	done := 0
	maxUID := uint32(0)

	var unsubscribed []act.Action
	var movedToJunk []act.Action

	aiClient := classify.AIClient{
		URL:   cfg.AIEndpoint,
		Key:   cfg.AIAPIKey,
		Model: cfg.AIModelId,
	}

	for _, record := range data {
		// slog.Info("record", "from", record.From, "subject", record.Subject, "date", record.Date.Format("Jan 02"),
		// 	"unsub_raw", helper.Trunc(record.UnsubRaw, 10))
		// mark := " "
		if record.UnsubRaw != "" {
			// mark = "U"
			record.ListUnsubscribe = mail.ParseListUnsub(record.UnsubRaw)
		}
		oneClick := strings.Contains(strings.ToLower(record.UnsubPost), "one-click")
		if record.UID > maxUID {
			maxUID = record.UID
		}
		rulesVerdict := classify.Classify(record)
		var aiVerdict classify.Verdict
		aiConfidence := 0.0
		if cfg.UseAI {
			aiVerdict, aiConfidence, err = aiClient.Classify(ctx, record)
			if err != nil {
				slog.Error("AI classify error", "uid", record.UID, "err", err)
			}
		}

		var a act.Action
		switch {
		case cfg.UseAI && rulesVerdict == aiVerdict:
			a = act.Plan(record, aiVerdict, oneClick)
		case cfg.UseAI && rulesVerdict != aiVerdict && aiConfidence >= 0.7:
			slog.Info("info", "uid", record.UID, "subject", record.Subject, "RulesVerdict", rulesVerdict, "AIVerdict", aiVerdict, "Confidence", strconv.FormatFloat(aiConfidence, 'f', -1, 64))
			a = act.Plan(record, aiVerdict, oneClick)
		default:
			a = act.Plan(record, rulesVerdict, oneClick)
		}

		if done >= *maxActions {
			slog.Info("action", "kind", "capped", "uid", a.UID)
			continue
		} else if *apply && a.Kind == "unsubscribe-oneclick" {
			if err := act.OneClick(ctx, a); err != nil {
				slog.Error("action", "uid", a.UID, "kind", a.Kind, "err", err)
				continue
			}
			unsubscribed = append(unsubscribed, a)

		}
		if *apply && (a.Kind == "unsubscribe-oneclick" || a.Kind == "move-to-junk") {
			if err := client.MoveToJunk(a.UID); err != nil {
				slog.Error("action", "uid", a.UID, "kind", a.Kind, "err", err)
				continue
			}
			movedToJunk = append(movedToJunk, a)
		}
		slog.Info("action", "uid", a.UID, "kind", a.Kind, "note", a.Note, "dry_run", !*apply)
		if a.Kind != "skip" {
			done++
			time.Sleep(500 * time.Millisecond)
		}
	}

	for _, a := range unsubscribed {
		st.UnsubscribeSent[a.UID] = true
	}
	for _, a := range movedToJunk {
		st.Junked[a.UID] = true
	}
	if maxUID > st.LastUID {
		st.LastUID = maxUID
	}
	if err := state.Save(cfg.StatePath, st); err != nil {
		slog.Error("state save", "err", err)
	}
	endTime := time.Now()
	slog.Info("info", "cull completed:", endTime.Format("15:04"))
	slog.Info("info", "duration:", endTime.Sub(startTime).String())
}
