package email

import (
	"fmt"
	"os"
	"strings"
)

// GenerateDotTrick creates all possible dot-trick variations for a given Gmail address
// and writes them to the specified output file. Returns the total number generated.
func GenerateDotTrick(rawEmail, outputFile string) (int, error) {
	parts := strings.Split(rawEmail, "@")
	if len(parts) != 2 || parts[1] != "gmail.com" {
		return 0, fmt.Errorf("must be a valid @gmail.com address")
	}

	username := parts[0]
	// Remove any existing dots
	username = strings.ReplaceAll(username, ".", "")
	domain := parts[1]

	n := len(username)
	if n <= 1 {
		return 0, fmt.Errorf("username too short")
	}

	// Calculate total combinations: 2^(n-1)
	totalCombos := 1 << (n - 1)

	file, err := os.Create(outputFile)
	if err != nil {
		return 0, fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	for i := 1; i < totalCombos; i++ {
		var variant strings.Builder
		variant.WriteByte(username[0])

		for j := 0; j < n-1; j++ {
			// Check if the j-th bit is set
			if (i & (1 << j)) != 0 {
				variant.WriteByte('.')
			}
			variant.WriteByte(username[j+1])
		}

		variant.WriteString("@" + domain)
		file.WriteString(variant.String() + "\n")
	}

	return totalCombos - 1, nil
}
