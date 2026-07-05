package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/verssache/chatgpt-creator/internal/config"
	"github.com/verssache/chatgpt-creator/internal/healthcheck"
	"github.com/verssache/chatgpt-creator/internal/register"
	"github.com/verssache/chatgpt-creator/internal/ui"
)

func main() {
	ui.ClearScreen()
	ui.PrintBanner()

	dataDir := flag.String("data", "data", "Data directory containing account files")
	proxy := flag.String("proxy", "", "Proxy to use for health checks")
	concurrent := flag.Int("workers", 5, "Number of concurrent health checks")
	autoRefresh := flag.Bool("refresh", false, "Auto-refresh expired tokens")
	flag.Parse()

	cfg, _ := config.Load("config.json")
	if cfg != nil && *proxy == "" {
		*proxy = cfg.Proxy
	}

	fmt.Println("\n" + ui.C("🔍 ACCOUNT HEALTH CHECKER", ui.Cyan))
	fmt.Println(strings.Repeat("─", 50))

	accounts, err := healthcheck.LoadAccounts(*dataDir)
	if err != nil {
		fmt.Printf(ui.C("⚠ Error loading accounts: %v\n", ui.Red), err)
		os.Exit(1)
	}

	if len(accounts) == 0 {
		fmt.Println(ui.C("⚠ No accounts found to check.", ui.Yellow))
		return
	}

	fmt.Printf("📦 Found %d accounts in %s/\n", len(accounts), *dataDir)
	fmt.Printf("🌐 Proxy: %s\n", *proxy)
	fmt.Printf("⚙️ Workers: %d\n", *concurrent)
	fmt.Printf("🔄 Auto-refresh: %v\n", *autoRefresh)
	fmt.Println()

	if *autoRefresh {
		fmt.Println(ui.C("⚠ Auto-refresh enabled. Updated tokens will be saved.", ui.Yellow))
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf(ui.C("Lanjutkan? (Y/n): ", ui.Yellow))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "n" {
			fmt.Println(ui.C("❌ Dibatalkan.", ui.Red))
			return
		}
	}

	var wg sync.WaitGroup
	workerCh := make(chan int, *concurrent)
	results := make([]*healthcheck.CheckResult, len(accounts))
	var mu sync.Mutex

	for i, token := range accounts {
		wg.Add(1)
		workerCh <- 1

		go func(idx int, tk *register.TokenResult) {
			defer wg.Done()
			defer func() { <-workerCh }()

			checker, err := healthcheck.NewChecker(*proxy)
			if err != nil {
				mu.Lock()
				results[idx] = &healthcheck.CheckResult{
					Email:  tk.Email,
					Status: healthcheck.StatusError,
					Error:  fmt.Sprintf("checker init failed: %v", err),
				}
				mu.Unlock()
				return
			}

			result := checker.CheckToken(tk)

			mu.Lock()
			results[idx] = result
			mu.Unlock()

			icon := "❓"
			switch result.Status {
			case healthcheck.StatusValid:
				icon = "✅"
			case healthcheck.StatusExpired:
				icon = "⚠️"
			case healthcheck.StatusRefreshed:
				icon = "🔄"
			case healthcheck.StatusError:
				icon = "❌"
			}

			fmt.Printf("  %s [%d/%d] %s\n", icon, idx+1, len(accounts), tk.Email)
		}(i, token)
	}

	wg.Wait()

	healthcheck.PrintSummary(results)

	if *autoRefresh {
		if err := healthcheck.SaveRefreshedTokens(*dataDir, results); err != nil {
			fmt.Printf(ui.C("⚠ Error saving refreshed tokens: %v\n", ui.Red), err)
		} else {
			refreshed := 0
			for _, r := range results {
				if r.Status == healthcheck.StatusRefreshed {
					refreshed++
				}
			}
			if refreshed > 0 {
				fmt.Printf(ui.C("✅ Refreshed %d tokens and saved to disk.\n", ui.Green), refreshed)
			}
		}
	}
}
