package main

import (
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

func main() {
	fmt.Println("Connecting to Gmail IMAP...")
	c, err := client.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		fmt.Println("Failed to connect:", err)
		return
	}
	defer c.Logout()

	if err := c.Login("davidexness1@gmail.com", "maej aftm hyvz ydqo"); err != nil {
		fmt.Println("Login failed:", err)
		return
	}

	_, err = c.Select("INBOX", false)
	if err != nil {
		fmt.Println("Failed to select INBOX:", err)
		return
	}

	// Search for recent emails from OpenAI
	criteria := imap.NewSearchCriteria()
	criteria.Since = time.Now().Add(-24 * time.Hour)
	uids, err := c.Search(criteria)
	if err != nil {
		fmt.Println("Search error:", err)
		return
	}

	if len(uids) == 0 {
		fmt.Println("No recent emails found.")
		return
	}

	if len(uids) > 10 {
		uids = uids[len(uids)-10:]
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	messages := make(chan *imap.Message, 100)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope, imap.FetchUid, imap.FetchFlags}

	go func() {
		c.Fetch(seqSet, items, messages)
	}()

	var allMessages []*imap.Message
	for msg := range messages {
		allMessages = append(allMessages, msg)
	}

	fmt.Printf("Found %d recent emails. Scanning newest to oldest...\n\n", len(allMessages))

	for i := len(allMessages) - 1; i >= 0; i-- {
		msg := allMessages[i]
		
		dateStr := msg.Envelope.Date.Format("15:04:05")
		subject := msg.Envelope.Subject
		
		flags := []string{}
		for _, f := range msg.Flags {
			flags = append(flags, f)
		}
		
		fmt.Printf("[%s] Subject: %s | Flags: %v\n", dateStr, subject, flags)
		
		// Check body for OTP
		foundOTP := false
		for _, body := range msg.Body {
			if body == nil { continue }
			mr, err := mail.CreateReader(body)
			if err != nil { continue }
			for {
				p, err := mr.NextPart()
				if err != nil { break }
				switch p.Header.(type) {
				case *mail.InlineHeader:
					b, _ := io.ReadAll(p.Body)
					bodyStr := string(b)
					re := regexp.MustCompile(`[^a-zA-Z0-9]([0-9]{6})[^a-zA-Z0-9]`)
					allMatches := re.FindAllStringSubmatch(bodyStr, -1)
					for _, m := range allMatches {
						if len(m) > 1 {
							code := m[1]
							if code != "202123" && code != "177010" && code != "353740" {
								fmt.Printf("   -> Extracted OTP: %s\n", code)
								foundOTP = true
							}
						}
					}
				}
			}
		}
		if !foundOTP {
			fmt.Printf("   -> NO OTP FOUND in this email.\n")
		}
		fmt.Println("--------------------------------------------------")
	}
}
