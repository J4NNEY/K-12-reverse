package manager

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/verssache/chatgpt-creator/internal/register"
)

// AccountEntry wraps a token result with metadata.
type AccountEntry struct {
	Token    *register.TokenResult
	Filename string
}

// LoadAllAccounts loads all accounts from the data directory.
func LoadAllAccounts(dataDir string) ([]*AccountEntry, error) {
	pattern := filepath.Join(dataDir, "accounts_*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		files, err = filepath.Glob("accounts.json")
		if err != nil {
			return nil, err
		}
	}

	var entries []*AccountEntry
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var tokens []*register.TokenResult
		if err := json.Unmarshal(data, &tokens); err != nil {
			continue
		}
		for _, t := range tokens {
			entries = append(entries, &AccountEntry{
				Token:    t,
				Filename: filepath.Base(f),
			})
		}
	}
	return entries, nil
}

// FilterAccounts filters accounts based on criteria.
type FilterCriteria struct {
	WorkspaceID string
	Email       string
	PlanType    string
}

func FilterAccounts(entries []*AccountEntry, criteria FilterCriteria) []*AccountEntry {
	if criteria.WorkspaceID == "" && criteria.Email == "" && criteria.PlanType == "" {
		return entries
	}

	var filtered []*AccountEntry
	for _, entry := range entries {
		token := entry.Token

		if criteria.WorkspaceID != "" {
			if _, ok := token.WorkspaceTokens[criteria.WorkspaceID]; !ok {
				continue
			}
		}

		if criteria.Email != "" {
			if !strings.Contains(strings.ToLower(token.Email), strings.ToLower(criteria.Email)) {
				continue
			}
		}

		filtered = append(filtered, entry)
	}
	return filtered
}

// ListAccounts prints a formatted list of accounts.
func ListAccounts(entries []*AccountEntry, showTokens bool) {
	if len(entries) == 0 {
		fmt.Println("No accounts found.")
		return
	}

	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("%-3s %-40s %-15s %-10s %s\n", "#", "Email", "Workspaces", "Source", "Token Preview")
	fmt.Println(strings.Repeat("─", 80))

	for i, entry := range entries {
		token := entry.Token
		wsCount := len(token.WorkspaceTokens)
		wsStr := fmt.Sprintf("%d workspace(s)", wsCount)
		if wsCount == 0 {
			wsStr = "free-tier"
		} else {
			var ids []string
			for id := range token.WorkspaceTokens {
				if len(id) > 8 {
					ids = append(ids, id[:8]+"...")
				} else {
					ids = append(ids, id)
				}
				if len(ids) >= 2 {
					break
				}
			}
			wsStr = strings.Join(ids, ", ")
			if wsCount > 2 {
				wsStr += fmt.Sprintf(" +%d more", wsCount-2)
			}
		}

		tokenPreview := "(hidden)"
		if showTokens {
			if len(token.AccessToken) > 40 {
				tokenPreview = token.AccessToken[:40] + "..."
			} else {
				tokenPreview = token.AccessToken
			}
		}

		fmt.Printf("%-3d %-40s %-15s %-10s %s\n",
			i+1,
			token.Email,
			wsStr,
			entry.Filename[:min(len(entry.Filename), 10)],
			tokenPreview,
		)
	}
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("Total: %d account(s)\n", len(entries))
}

// ExportAccounts exports accounts in the specified format.
func ExportAccounts(entries []*AccountEntry, format string, outputFile string) error {
	switch strings.ToLower(format) {
	case "json":
		return exportJSON(entries, outputFile)
	case "csv":
		return exportCSV(entries, outputFile)
	case "txt":
		return exportTXT(entries, outputFile)
	default:
		return fmt.Errorf("unsupported format: %s (use json, csv, or txt)", format)
	}
}

func exportJSON(entries []*AccountEntry, outputFile string) error {
	tokens := make([]*register.TokenResult, len(entries))
	for i, entry := range entries {
		tokens[i] = entry.Token
	}

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputFile, data, 0644)
}

func exportCSV(entries []*AccountEntry, outputFile string) error {
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	writer.Write([]string{"Email", "Password", "AccessToken", "RefreshToken", "WorkspaceIDs", "SourceFile"})

	for _, entry := range entries {
		token := entry.Token
		var wsIDs []string
		for id := range token.WorkspaceTokens {
			wsIDs = append(wsIDs, id)
		}

		writer.Write([]string{
			token.Email,
			token.Password,
			token.AccessToken,
			token.RefreshToken,
			strings.Join(wsIDs, ";"),
			entry.Filename,
		})
	}

	return nil
}

func exportTXT(entries []*AccountEntry, outputFile string) error {
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, entry := range entries {
		token := entry.Token
		line := fmt.Sprintf("%s|%s|%s", token.Email, token.Password, token.AccessToken)

		if len(token.WorkspaceTokens) > 0 {
			var wsIDs []string
			for id := range token.WorkspaceTokens {
				wsIDs = append(wsIDs, id)
			}
			line += "|" + strings.Join(wsIDs, ",")
		}

		line += "\n"
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// DeleteAccount removes an account from its source file based on email.
func DeleteAccount(entries []*AccountEntry, email string, dataDir string) (bool, error) {
	emailLower := strings.ToLower(strings.TrimSpace(email))

	for _, entry := range entries {
		if strings.EqualFold(entry.Token.Email, emailLower) {
			filename := entry.Filename
			filePath := filepath.Join(dataDir, filename)

			existingData, err := os.ReadFile(filePath)
			if err != nil {
				return false, fmt.Errorf("cannot read source file %s: %w", filename, err)
			}

			var allTokens []*register.TokenResult
			json.Unmarshal(existingData, &allTokens)

			var updated []*register.TokenResult
			found := false
			for _, t := range allTokens {
				if !strings.EqualFold(t.Email, emailLower) {
					updated = append(updated, t)
				} else {
					found = true
				}
			}

			if !found {
				return false, fmt.Errorf("email %s not found in %s", email, filename)
			}

			if len(updated) == 0 {
				os.Remove(filePath)
				return true, nil
			}

			newData, _ := json.MarshalIndent(updated, "", "  ")
			if err := os.WriteFile(filePath, newData, 0644); err != nil {
				return false, fmt.Errorf("failed to write updated file: %w", err)
			}

			return true, nil
		}
	}

	return false, fmt.Errorf("email %s not found in any account file", email)
}

// Stats returns a string summarizing account stats.
func Stats(entries []*AccountEntry) string {
	total := len(entries)
	wsCount := 0
	freeCount := 0

	for _, entry := range entries {
		if len(entry.Token.WorkspaceTokens) > 0 {
			wsCount++
		} else {
			freeCount++
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("📊 Account Stats (%s)\n", time.Now().Format("2006-01-02 15:04")))
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Total accounts : %d\n", total))
	b.WriteString(fmt.Sprintf("  K12 workspace  : %d\n", wsCount))
	b.WriteString(fmt.Sprintf("  Free tier      : %d\n", freeCount))
	b.WriteString(strings.Repeat("─", 40))
	return b.String()
}
