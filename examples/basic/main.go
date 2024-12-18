package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/iamlongalong/readgo"
)

func main() {
	// Create a new analyzer with basic options
	analyzer := readgo.NewAnalyzer(
		readgo.WithWorkDir("."),
		readgo.WithCacheTTL(5*time.Minute),
	)

	// Analyze the current project
	result, err := analyzer.AnalyzeProject(context.Background(), ".")
	if err != nil {
		log.Fatalf("Failed to analyze project: %v", err)
	}

	// Print project information
	fmt.Printf("Project: %s\n", result.Name)
	fmt.Printf("Path: %s\n", result.Path)
	fmt.Printf("Analyzed at: %s\n\n", result.AnalyzedAt)

	// Print imports
	fmt.Println("Imports:")
	for _, imp := range result.Imports {
		fmt.Printf("  - %s\n", imp)
	}
	fmt.Println()

	// Print types
	fmt.Println("Types:")
	for _, t := range result.Types {
		fmt.Printf("  - %s.%s: %s\n", t.Package, t.Name, t.Type)
	}
	fmt.Println()

	// Print functions
	fmt.Println("Functions:")
	for _, f := range result.Functions {
		fmt.Printf("  - %s.%s\n", f.Package, f.Name)
	}

	// Print cache statistics
	fmt.Println("\nCache Statistics:")
	fmt.Println("----------------")
	stats := analyzer.GetCacheStats()
	for key, value := range stats {
		fmt.Printf("%s: %v\n", key, value)
	}
}
