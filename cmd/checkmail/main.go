package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

func main() {
	base := strings.TrimSpace(os.Getenv("GMAIL_BASE"))
	pass := strings.TrimSpace(os.Getenv("GMAIL_APP_PASSWORD"))
	if base == "" || pass == "" {
		fmt.Fprintln(os.Stderr, "set GMAIL_BASE and GMAIL_APP_PASSWORD")
		os.Exit(1)
	}

	c, err := client.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "imap connect failed: %v\n", err)
		os.Exit(1)
	}
	defer c.Logout()

	if err := c.Login(base, pass); err != nil {
		fmt.Fprintf(os.Stderr, "imap login failed: %v\n", err)
		os.Exit(1)
	}

	mbox, err := c.Select("INBOX", true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "select inbox failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("IMAP login OK: %s (%d messages)\n", base, mbox.Messages)
	if mbox.Messages == 0 {
		return
	}

	from := uint32(1)
	if mbox.Messages > 10 {
		from = mbox.Messages - 9
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(from, mbox.Messages)
	messages := make(chan *imap.Message, 10)
	go func() {
		_ = c.Fetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags}, messages)
	}()

	var rows []*imap.Message
	for msg := range messages {
		if msg != nil && msg.Envelope != nil {
			rows = append(rows, msg)
		}
	}

	for i := len(rows) - 1; i >= 0; i-- {
		msg := rows[i]
		fromAddr := "unknown"
		if len(msg.Envelope.From) > 0 {
			a := msg.Envelope.From[0]
			fromAddr = a.MailboxName + "@" + a.HostName
		}
		seen := "unread"
		for _, flag := range msg.Flags {
			if strings.EqualFold(flag, imap.SeenFlag) {
				seen = "seen"
				break
			}
		}
		fmt.Printf("%s | %s | %s | %s\n", msg.Envelope.Date.Format(time.RFC3339), seen, fromAddr, msg.Envelope.Subject)
	}
}
