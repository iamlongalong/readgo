package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/iamlongalong/readgo"
)

func main() {
	// Create a new analyzer with options
	analyzer := readgo.NewAnalyzer(
		readgo.WithWorkDir("."),
		readgo.WithCacheTTL(5*time.Minute),
		readgo.WithAnalysisTimeout(10*time.Second),
	)

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
		if strings.Contains(t.Type, "interface") {
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

	// Print cache statistics
	fmt.Println("Cache Statistics:")
	fmt.Println("----------------")
	stats := analyzer.GetCacheStats()
	for key, value := range stats {
		fmt.Printf("%s: %v\n", key, value)
	}
}
