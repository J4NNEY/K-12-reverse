package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <email> <output_file>")
		fmt.Println("Example: go run main.go davidexness1@gmail.com mylist.txt")
		return
	}

	rawEmail := os.Args[1]
	outputFile := os.Args[2]

	parts := strings.Split(rawEmail, "@")
	if len(parts) != 2 || parts[1] != "gmail.com" {
		fmt.Println("Error: Must be a valid @gmail.com address")
		return
	}

	username := parts[0]
	// Remove any existing dots
	username = strings.ReplaceAll(username, ".", "")
	domain := parts[1]

	n := len(username)
	if n <= 1 {
		fmt.Println("Username too short")
		return
	}

	// Calculate total combinations: 2^(n-1)
	totalCombos := 1 << (n - 1)
	
	fmt.Printf("Generating %d dot-trick variations for %s...\n", totalCombos, rawEmail)

	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	for i := 0; i < totalCombos; i++ {
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

	fmt.Printf("✅ Successfully saved %d variations to %s\n", totalCombos, outputFile)
}
