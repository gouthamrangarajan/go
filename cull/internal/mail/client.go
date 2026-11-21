package mail

import (
	"bytes"
	"cull/internal/config"
	"cull/internal/state"
	"fmt"
	"log"
	"log/slog"
	"mime"
	"net/mail"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

type Header struct {
	Seq             uint32
	UID             uint32
	From            string
	Subject         string
	Date            time.Time
	UnsubRaw        string
	ListUnsubscribe Unsub
	UnsubPost       string
}

type Client struct {
	c              *imapclient.Client
	st             *state.State
	numMessages    uint32
	junkFolderName string
}

func Open(config config.Config, st *state.State) (*Client, error) { // dial, login, select, validate UIDVALIDITY
	conn, err := imapclient.DialTLS(config.IMAPHost+":"+config.IMAPPort, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	if err := conn.Login(config.IMAPUser, config.IMAPPass).Wait(); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	sel, err := conn.Select("INBOX", nil).Wait()
	if err != nil {
		_ = conn.Logout().Wait()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}
	if st.UIDValidity != 0 && st.UIDValidity != sel.UIDValidity {
		slog.Warn("uidvalidity changed, resetting state",
			"old", st.UIDValidity, "new", sel.UIDValidity)
		*st = state.State{UIDValidity: sel.UIDValidity}
	} else {
		st.UIDValidity = sel.UIDValidity
	}

	slog.Info("connected", "inbox_messages", sel.NumMessages, "uidvalidity", sel.UIDValidity)
	return &Client{c: conn, st: st, numMessages: sel.NumMessages, junkFolderName: "Bulk"}, nil
}
func (cl *Client) RecentHeaders(limit int) ([]Header, error) {
	if cl.numMessages == 0 {
		return nil, nil
	}

	var numSet imap.NumSet
	if cl.st.LastUID == 0 {
		start := uint32(1)
		if cl.numMessages > uint32(limit) {
			start = cl.numMessages - uint32(limit) + 1
		}

		var seqSet imap.SeqSet
		seqSet.AddRange(start, cl.numMessages)
		numSet = seqSet
	} else {
		var uiSet imap.UIDSet
		uiSet.AddRange(imap.UID(cl.st.LastUID+1), 0)
		numSet = uiSet
	}

	cmd := cl.c.Fetch(numSet, &imap.FetchOptions{
		UID: true,
		BodySection: []*imap.FetchItemBodySection{{
			Specifier: imap.PartSpecifierHeader,
			Peek:      true,
		}},
	})
	msgs, err := cmd.Collect()
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	out := make([]Header, 0, len(msgs))
	for _, msg := range msgs {
		uid := uint32(msg.UID)
		if uid <= cl.st.LastUID || cl.st.UnsubscribeSent[uid] {
			continue // already picked in a prior run — don't re-POST
		}
		h := parseHeader(msg)
		out = append(out, h)
	}
	return out, nil
}

func (cl *Client) Close() error {
	_ = cl.c.Logout().Wait()
	return cl.c.Close()
}
func (cl *Client) Folders() ([]string, error) {
	data, err := cl.c.List("", "*", nil).Collect()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(data))
	for _, d := range data {
		out = append(out, d.Mailbox)
	}
	return out, nil
}
func (cl *Client) CheckBulkAndJunk() {
	for _, name := range []string{"Bulk", "Junk"} {
		sel, err := cl.c.Select(name, nil).Wait()
		if err != nil {
			slog.Error("select", "name", name, "err", err)
			continue
		}
		slog.Info("folder-probe", "name", name, "messages", sel.NumMessages, "uidvalidity", sel.UIDValidity)
	}

}
func (cl *Client) SelectInbox() (*imap.SelectData, error) {
	if sel, err := cl.c.Select("INBOX", nil).Wait(); err != nil {
		return nil, fmt.Errorf("re-select INBOX: %w", err)
	} else {
		return sel, nil
	}

}
func (cl *Client) MoveToJunk(uid uint32) error {
	var set imap.UIDSet
	set.AddNum(imap.UID(uid))

	if _, err := cl.c.Move(set, cl.junkFolderName).Wait(); err != nil {
		return fmt.Errorf("move uid %d: %w", uid, err)
	}
	return nil
}

func parseHeader(msg *imapclient.FetchMessageBuffer) Header {
	h := Header{Seq: msg.SeqNum, UID: uint32(msg.UID)}
	for _, bs := range msg.BodySection {
		m, perr := mail.ReadMessage(bytes.NewReader(bs.Bytes))
		if perr != nil {
			log.Printf("parse seq %d: %v", msg.SeqNum, perr)
			continue
		}
		h.From = cleanFrom(m.Header.Get("From"))
		h.Subject = decodeHeader(m.Header.Get("Subject"))
		h.UnsubRaw = m.Header.Get("List-Unsubscribe")
		h.UnsubPost = m.Header.Get("List-Unsubscribe-Post")
		h.Date, _ = m.Header.Date()
		break
	}
	return h
}

func cleanFrom(raw string) string {
	if raw == "" {
		return ""
	}
	if a, err := mail.ParseAddress(raw); err == nil {
		return a.Address
	}
	return raw
}

func decodeHeader(s string) string {
	if d, err := new(mime.WordDecoder).DecodeHeader(s); err == nil {
		return d
	}
	return s
}
