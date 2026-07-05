package email

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

// GetK12InviteLinkViaIMAP connects to Gmail IMAP and retrieves the K12 invite link.
func GetK12InviteLinkViaIMAP(cfg GmailIMAPConfig, targetEmail string, maxRetries int, delay time.Duration) (string, error) {
	for i := 0; i < maxRetries; i++ {
		link, err := fetchInviteLinkFromGmail(cfg)
		if err == nil && link != "" {
			return link, nil
		}
		time.Sleep(delay)
	}

	return "", fmt.Errorf("failed to get invite link from Gmail after %d retries", maxRetries)
}

func fetchInviteLinkFromGmail(cfg GmailIMAPConfig) (string, error) {
	c, err := client.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		return "", fmt.Errorf("failed to connect to IMAP: %w", err)
	}
	defer c.Logout()

	if err := c.Login(cfg.Email, cfg.AppPassword); err != nil {
		return "", fmt.Errorf("IMAP login failed: %w", err)
	}

	_, err = c.Select("INBOX", false)
	if err != nil {
		return "", fmt.Errorf("failed to select INBOX: %w", err)
	}

	// Search for UNSEEN emails
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	uids, err := c.Search(criteria)
	if err != nil {
		return "", fmt.Errorf("search error: %w", err)
	}
	if len(uids) == 0 {
		return "", fmt.Errorf("no unread emails")
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	messages := make(chan *imap.Message, 100)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope, imap.FetchUid, imap.FetchFlags}

	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqSet, items, messages)
	}()

	var allMessages []*imap.Message
	for msg := range messages {
		allMessages = append(allMessages, msg)
	}

	for i := len(allMessages) - 1; i >= 0; i-- {
		msg := allMessages[i]
		if msg == nil || msg.Envelope == nil {
			continue
		}

		subject := msg.Envelope.Subject
		if !strings.Contains(strings.ToLower(subject), "request to join") && !strings.Contains(strings.ToLower(subject), "workspace") {
			continue
		}

		for _, body := range msg.Body {
			if body == nil {
				continue
			}

			mr, err := mail.CreateReader(body)
			if err != nil {
				continue
			}

			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				} else if err != nil {
					break
				}

				switch p.Header.(type) {
				case *mail.InlineHeader:
					b, err := io.ReadAll(p.Body)
					if err != nil {
						continue
					}
					bodyStr := string(b)

					re := regexp.MustCompile(`https://chatgpt\.com/k12-invite\?[^\s"'>]+`)
					match := re.FindString(bodyStr)
					if match != "" {
						// Mark as read
						item := imap.FormatFlagsOp(imap.AddFlags, true)
						flags := []interface{}{imap.SeenFlag}
						seq := new(imap.SeqSet)
						seq.AddNum(msg.SeqNum)
						c.Store(seq, item, flags, nil)

						return match, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no invite link found")
}
