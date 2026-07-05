package email

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

type PoolItem struct {
	Email    string
	ListFile string
}

// GmailDotPool manages a pool of Gmail dot-trick email addresses across multiple accounts.
type GmailDotPool struct {
	items []PoolItem
	index int
	mu    sync.Mutex
}

// NewMultiGmailPool creates a single pool from multiple list files using Round-Robin distribution.
func NewMultiGmailPool(listFiles []string) (*GmailDotPool, error) {
	var lists [][]PoolItem

	for _, filePath := range listFiles {
		file, err := os.Open(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // skip missing files, handled by auto-generator
			}
			return nil, fmt.Errorf("failed to open Gmail list file %s: %w", filePath, err)
		}

		var currentList []PoolItem
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && strings.Contains(line, "@") {
				currentList = append(currentList, PoolItem{
					Email:    line,
					ListFile: filePath,
				})
			}
		}
		file.Close()

		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("error reading Gmail list file %s: %w", filePath, err)
		}

		if len(currentList) > 0 {
			// Shuffle the current list so dot-trick and plus-trick variations rotate randomly
			rand.Seed(time.Now().UnixNano())
			rand.Shuffle(len(currentList), func(i, j int) {
				currentList[i], currentList[j] = currentList[j], currentList[i]
			})
			lists = append(lists, currentList)
		}
	}

	// Interleave (Round-Robin)
	var items []PoolItem
	for {
		added := false
		for i := 0; i < len(lists); i++ {
			if len(lists[i]) > 0 {
				items = append(items, lists[i][0])
				lists[i] = lists[i][1:] // pop first element
				added = true
			}
		}
		if !added {
			break // all lists are exhausted
		}
	}

	return &GmailDotPool{
		items: items,
		index: 0,
	}, nil
}

// Next returns the next available Gmail address from the pool.
func (p *GmailDotPool) Next() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.index >= len(p.items) {
		return "", fmt.Errorf("all %d Gmail addresses have been used", len(p.items))
	}

	item := p.items[p.index]
	p.index++
	return item.Email, nil
}

// Remaining returns how many unused email addresses are left.
func (p *GmailDotPool) Remaining() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.items) - p.index
}

// Total returns the total number of email addresses in the pool.
func (p *GmailDotPool) Total() int {
	return len(p.items)
}

// MarkConsumed removes the email from its corresponding text file.
// This ensures that the list shrinks when an account succeeds or is marked as a zombie.
func (p *GmailDotPool) MarkConsumed(email string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var targetFile string
	for _, item := range p.items {
		if strings.EqualFold(item.Email, email) {
			targetFile = item.ListFile
			break
		}
	}

	if targetFile == "" {
		return nil // Email not found in pool tracker
	}

	// Read the file, remove the email, rewrite
	file, err := os.Open(targetFile)
	if err != nil {
		return err
	}

	var remaining []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.EqualFold(line, email) && line != "" {
			remaining = append(remaining, line)
		}
	}
	file.Close()

	// Rewrite file
	outFile, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	for _, line := range remaining {
		outFile.WriteString(line + "\n")
	}

	return nil
}
