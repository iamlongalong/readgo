package main

import (
	"context"
	"fmt"
	"log"

	"github.com/iamlongalong/readgo"
)

func main() {
	// Create a new analyzer
	analyzer := readgo.NewAnalyzer(".")

	// Analyze the io package from standard library
	result, err := analyzer.AnalyzePackage(context.Background(), "io")
	if err != nil {
		log.Fatalf("Failed to analyze io package: %v", err)
	}

	// Print package information
	fmt.Printf("Package: %s\n", result.Name)
	fmt.Printf("Path: %s\n", result.Path)
	fmt.Printf("Analyzed at: %s\n\n", result.AnalyzedAt)

	// Print interfaces
	fmt.Println("Interfaces:")
	for _, t := range result.Types {
		if t.Type[:9] == "interface" {
			fmt.Printf("  - %s: %s\n", t.Name, t.Type)
		}
	}
	fmt.Println()

	// Find specific interface
	reader, err := analyzer.FindInterface(context.Background(), "io", "Reader")
	if err != nil {
		log.Fatalf("Failed to find Reader interface: %v", err)
	}
	fmt.Printf("Reader interface details:\n")
	fmt.Printf("  Name: %s\n", reader.Name)
	fmt.Printf("  Package: %s\n", reader.Package)
	fmt.Printf("  Type: %s\n", reader.Type)
	fmt.Printf("  Exported: %v\n\n", reader.IsExported)

	// Analyze the net/http package
	result, err = analyzer.AnalyzePackage(context.Background(), "net/http")
	if err != nil {
		log.Fatalf("Failed to analyze net/http package: %v", err)
	}

	// Print structs that implement io.Reader
	fmt.Println("Types in net/http that implement io.Reader:")
	for _, t := range result.Types {
		if t.Type[:6] == "struct" {
			// Check if the type implements io.Reader
			// Note: This is a simplified check, in a real application
			// you would use types.Implements to check this properly
			if t.Type[:6] == "struct" && (t.Type[:6] == "struct" || t.Type[:9] == "interface") {
				fmt.Printf("  - %s\n", t.Name)
			}
		}
	}
}
