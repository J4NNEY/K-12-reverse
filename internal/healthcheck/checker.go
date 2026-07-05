package healthcheck

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	http "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/tls-client"
	"github.com/verssache/chatgpt-creator/internal/chrome"
	"github.com/verssache/chatgpt-creator/internal/register"
	"github.com/verssache/chatgpt-creator/internal/util"
)

const (
	baseURL = "https://chatgpt.com"
	authURL = "https://auth.openai.com"
)

type AccountStatus string

const (
	StatusValid     AccountStatus = "valid"
	StatusExpired   AccountStatus = "expired"
	StatusError     AccountStatus = "error"
	StatusRefreshed AccountStatus = "refreshed"
)

type CheckResult struct {
	Email        string        `json:"email"`
	Status       AccountStatus `json:"status"`
	AccessToken  string        `json:"accessToken"`
	RefreshToken string        `json:"refreshToken"`
	Password     string        `json:"password"`
	WorkspaceIDs []string      `json:"workspaceIds,omitempty"`
	Error        string        `json:"error,omitempty"`
	PlanType     string        `json:"planType,omitempty"`
}

type Checker struct {
	proxy     string
	client    tls_client.HttpClient
	userAgent string
	secChUA   string
	mu        sync.Mutex
}

func NewChecker(proxy string) (*Checker, error) {
	profile, _, ua := chrome.RandomChromeVersion()
	mappedProfile := chrome.MapToTLSProfile(profile.Impersonate)

	options := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(mappedProfile),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
	}

	if proxy != "" {
		formattedProxy := util.FormatProxy(proxy)
		options = append(options, tls_client.WithProxyUrl(formattedProxy))
	}

	session, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}

	return &Checker{
		proxy:     proxy,
		client:    session,
		userAgent: ua,
		secChUA:   profile.SecChUA,
	}, nil
}

func (c *Checker) do(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	}
	if req.Header.Get("sec-ch-ua") == "" {
		req.Header.Set("sec-ch-ua", c.secChUA)
	}
	if req.Header.Get("sec-ch-ua-mobile") == "" {
		req.Header.Set("sec-ch-ua-mobile", "?0")
	}
	if req.Header.Get("sec-ch-ua-platform") == "" {
		req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	}
	return c.client.Do(req)
}

type sessionResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	IdToken      string `json:"idToken"`
	Expires      string `json:"expires"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

func (c *Checker) CheckToken(token *register.TokenResult) *CheckResult {
	result := &CheckResult{
		Email:        token.Email,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Password:     token.Password,
	}

	for wsID := range token.WorkspaceTokens {
		result.WorkspaceIDs = append(result.WorkspaceIDs, wsID)
	}

	visitReq, _ := http.NewRequest("GET", baseURL+"/", nil)
	visitReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	visitReq.Header.Set("Upgrade-Insecure-Requests", "1")
	resp, err := c.do(visitReq)
	if err != nil {
		result.Status = StatusError
		result.Error = fmt.Sprintf("visit failed: %v", err)
		return result
	}
	resp.Body.Close()

	req, _ := http.NewRequest("GET", baseURL+"/api/auth/session", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Referer", baseURL+"/")

	resp, err = c.do(req)
	if err != nil {
		result.Status = StatusError
		result.Error = fmt.Sprintf("session request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		var session sessionResponse
		if err := json.Unmarshal(body, &session); err != nil {
			result.Status = StatusError
			result.Error = fmt.Sprintf("parse failed: %v", err)
			return result
		}

		result.Status = StatusValid
		result.AccessToken = session.AccessToken

		if session.Expires != "" {
			if expTime, err := time.Parse(time.RFC3339, session.Expires); err == nil {
				if time.Until(expTime) < 24*time.Hour {
					result.Status = StatusExpired
					result.Error = fmt.Sprintf("expires soon: %s", session.Expires)
				}
			}
		}

		return result
	}

	if resp.StatusCode == 401 {
		refreshResult := c.tryRefresh(token)
		if refreshResult != nil {
			return refreshResult
		}

		result.Status = StatusExpired
		result.Error = "token expired and refresh failed"
		return result
	}

	result.Status = StatusError
	result.Error = fmt.Sprintf("unexpected status: %d", resp.StatusCode)
	return result
}

type refreshPayload struct {
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	GrantType    string `json:"grant_type"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (c *Checker) tryRefresh(token *register.TokenResult) *CheckResult {
	if token.RefreshToken == "" || token.RefreshToken == "not available" {
		return nil
	}

	payload := refreshPayload{
		RefreshToken: token.RefreshToken,
		ClientID:     "app_X8zY6v2PpQ9tR3dE7nK1jL5gH",
		GrantType:    "refresh_token",
	}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://auth.openai.com/oauth/token", strings.NewReader(string(jsonPayload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil
	}

	var refreshResp refreshResponse
	if err := json.Unmarshal(body, &refreshResp); err != nil {
		return nil
	}

	result := &CheckResult{
		Email:        token.Email,
		Status:       StatusRefreshed,
		AccessToken:  refreshResp.AccessToken,
		RefreshToken: refreshResp.RefreshToken,
		Password:     token.Password,
	}

	for wsID := range token.WorkspaceTokens {
		result.WorkspaceIDs = append(result.WorkspaceIDs, wsID)
	}

	return result
}

// LoadAccounts reads all account token files from the given directory.
func LoadAccounts(dataDir string) ([]*register.TokenResult, error) {
	pattern := filepath.Join(dataDir, "accounts_*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list account files: %w", err)
	}

	if len(files) == 0 {
		pattern = filepath.Join(dataDir, "accounts.json")
		files, err = filepath.Glob(pattern)
		if err != nil || len(files) == 0 {
			dir := "."
			pattern = "accounts.json"
			files, err = filepath.Glob(filepath.Join(dir, pattern))
			if err != nil || len(files) == 0 {
				return nil, fmt.Errorf("no account files found in %s", dataDir)
			}
		}
	}

	var allTokens []*register.TokenResult
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var tokens []*register.TokenResult
		if err := json.Unmarshal(data, &tokens); err != nil {
			continue
		}
		allTokens = append(allTokens, tokens...)
	}

	return allTokens, nil
}

// SaveRefreshedTokens saves updated tokens back to the original files.
func SaveRefreshedTokens(dataDir string, results []*CheckResult) error {
	updated := make(map[string][]*register.TokenResult)

	for _, r := range results {
		if r.Status != StatusRefreshed {
			continue
		}

		emailLower := strings.ToLower(r.Email)
		baseEmail := getBaseEmail(emailLower)
		username := strings.Split(baseEmail, "@")[0]

		token := &register.TokenResult{
			AccessToken:  r.AccessToken,
			RefreshToken: r.RefreshToken,
			IdToken:      r.AccessToken,
			Email:        r.Email,
			Password:     r.Password,
		}

		updated[username] = append(updated[username], token)
	}

	for username, tokens := range updated {
		tokenFile := filepath.Join(dataDir, fmt.Sprintf("accounts_%s.json", username))
		data, err := json.MarshalIndent(tokens, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal tokens for %s: %w", username, err)
		}
		if err := os.WriteFile(tokenFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write tokens for %s: %w", username, err)
		}
	}

	return nil
}

func getBaseEmail(emailAddr string) string {
	parts := strings.Split(emailAddr, "@")
	if len(parts) != 2 {
		return emailAddr
	}
	return strings.ReplaceAll(parts[0], ".", "") + "@" + parts[1]
}

func PrintSummary(results []*CheckResult) {
	valid := 0
	expired := 0
	refreshed := 0
	errCount := 0

	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("📊  ACCOUNT HEALTH CHECK RESULTS")
	fmt.Println(strings.Repeat("═", 60))

	for _, r := range results {
		icon := "❓"
		switch r.Status {
		case StatusValid:
			icon = "✅"
			valid++
		case StatusExpired:
			icon = "⚠️"
			expired++
		case StatusRefreshed:
			icon = "🔄"
			refreshed++
		case StatusError:
			icon = "❌"
			errCount++
		}

		planInfo := ""
		if r.PlanType != "" {
			planInfo = fmt.Sprintf(" [%s]", r.PlanType)
		}

		fmt.Printf("  %s %s%s\n", icon, r.Email, planInfo)
		if r.Error != "" {
			fmt.Printf("     └─ %s\n", r.Error)
		}
	}

	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("  ✅ Valid:     %d\n", valid)
	fmt.Printf("  🔄 Refreshed: %d\n", refreshed)
	fmt.Printf("  ⚠️ Expired:   %d\n", expired)
	fmt.Printf("  ❌ Error:     %d\n", errCount)
	fmt.Printf("  📦 Total:     %d\n", len(results))
	fmt.Println(strings.Repeat("═", 60))
}
