package ui

import "fmt"

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
	Bold   = "\033[1m"
)

// C formats text with a color and bold style.
func C(text, color string) string {
	return fmt.Sprintf("%s%s%s%s", Bold, color, text, Reset)
}

// PrintBanner prints the main application banner.
func PrintBanner() {
	banner := `
╔══════════════════════════════════════════════════╗
║             K12-REVERSE BY AHMADD4VD             ║
║      Advanced ChatGPT Account Registration       ║
╚══════════════════════════════════════════════════╝
`
	fmt.Print(C(banner, Cyan))
}

// ClearScreen clears the terminal screen using ANSI escape codes.
func ClearScreen() {
	fmt.Print("\033[H\033[2J")
}
