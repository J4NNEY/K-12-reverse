package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/verssache/chatgpt-creator/internal/manager"
	"github.com/verssache/chatgpt-creator/internal/ui"
)

func main() {
	ui.ClearScreen()
	ui.PrintBanner()

	dataDir := flag.String("data", "data", "Data directory containing account files")
	showTokens := flag.Bool("show-tokens", false, "Show access tokens in output")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		return
	}

	command := args[0]

	switch command {
	case "list":
		cmdList(*dataDir, *showTokens, args[1:])
	case "export":
		cmdExport(*dataDir, args[1:])
	case "delete":
		cmdDelete(*dataDir, args[1:])
	case "stats":
		cmdStats(*dataDir)
	case "help":
		printUsage()
	default:
		fmt.Printf(ui.C("Unknown command: %s\n\n", ui.Red), command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println(ui.C("\n📋 ACCOUNT MANAGER", ui.Cyan))
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println("Usage: go run cmd/manager/main.go [options] <command> [args]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --data <dir>         Data directory (default: data/)")
	fmt.Println("  --show-tokens        Show access tokens in output")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list [--workspace <id>] [--email <query>]")
	fmt.Println("       List all accounts, optionally filtered")
	fmt.Println()
	fmt.Println("  export <format> [output]")
	fmt.Println("       Export accounts (json|txt|csv)")
	fmt.Println("       Default output: export_tokens.<format>")
	fmt.Println()
	fmt.Println("  delete <email>")
	fmt.Println("       Delete an account by email")
	fmt.Println()
	fmt.Println("  stats")
	fmt.Println("       Show account statistics")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/manager/main.go list")
	fmt.Println("  go run cmd/manager/main.go list --workspace ff598c4d")
	fmt.Println("  go run cmd/manager/main.go export json")
	fmt.Println("  go run cmd/manager/main.go export csv ./backup.csv")
	fmt.Println("  go run cmd/manager/main.go delete test@gmail.com")
	fmt.Println("  go run cmd/manager/main.go stats")
	fmt.Println(strings.Repeat("─", 50))
}

func cmdList(dataDir string, showTokens bool, args []string) {
	workspaceID := ""
	email := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--workspace":
			if i+1 < len(args) {
				workspaceID = args[i+1]
				i++
			}
		case "--email":
			if i+1 < len(args) {
				email = args[i+1]
				i++
			}
		}
	}

	entries, err := manager.LoadAllAccounts(dataDir)
	if err != nil {
		fmt.Printf(ui.C("⚠ Error loading accounts: %v\n", ui.Red), err)
		return
	}

	criteria := manager.FilterCriteria{
		WorkspaceID: workspaceID,
		Email:       email,
	}
	filtered := manager.FilterAccounts(entries, criteria)

	manager.ListAccounts(filtered, showTokens)
}

func cmdExport(dataDir string, args []string) {
	if len(args) == 0 {
		fmt.Println(ui.C("⚠ Usage: export <format> [output]", ui.Red))
		return
	}

	format := args[0]
	outputFile := fmt.Sprintf("export_tokens.%s", format)
	if len(args) > 1 {
		outputFile = args[1]
	}

	entries, err := manager.LoadAllAccounts(dataDir)
	if err != nil {
		fmt.Printf(ui.C("⚠ Error loading accounts: %v\n", ui.Red), err)
		return
	}

	if err := manager.ExportAccounts(entries, format, outputFile); err != nil {
		fmt.Printf(ui.C("⚠ Export failed: %v\n", ui.Red), err)
		return
	}

	fmt.Printf(ui.C("✅ Exported %d accounts to %s (format: %s)\n", ui.Green), len(entries), outputFile, format)
}

func cmdDelete(dataDir string, args []string) {
	if len(args) == 0 {
		fmt.Println(ui.C("⚠ Usage: delete <email>", ui.Red))
		return
	}

	email := args[0]

	entries, err := manager.LoadAllAccounts(dataDir)
	if err != nil {
		fmt.Printf(ui.C("⚠ Error loading accounts: %v\n", ui.Red), err)
		return
	}

	deleted, err := manager.DeleteAccount(entries, email, dataDir)
	if err != nil {
		fmt.Printf(ui.C("⚠ Delete failed: %v\n", ui.Red), err)
		return
	}

	if deleted {
		fmt.Printf(ui.C("✅ Deleted account: %s\n", ui.Green), email)
	}
}

func cmdStats(dataDir string) {
	entries, err := manager.LoadAllAccounts(dataDir)
	if err != nil {
		fmt.Printf(ui.C("⚠ Error loading accounts: %v\n", ui.Red), err)
		return
	}
	fmt.Println(manager.Stats(entries))
}
