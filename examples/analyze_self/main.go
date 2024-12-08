package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/iamlongalong/readgo"
)

func main() {
	// Create a new analyzer
	analyzer := readgo.NewAnalyzer(".")

	// First analyze the entire project
	fmt.Println("Analyzing entire project:")
	fmt.Println(strings.Repeat("=", 80))
	analyzeProject(analyzer)

	// Then analyze specific interfaces
	fmt.Println("\nAnalyzing core interfaces:")
	fmt.Println(strings.Repeat("=", 80))
	analyzeInterfaces(analyzer)

	// Finally analyze implementations
	fmt.Println("\nAnalyzing implementations:")
	fmt.Println(strings.Repeat("=", 80))
	analyzeImplementations(analyzer)
}

func analyzeProject(analyzer *readgo.DefaultAnalyzer) {
	result, err := analyzer.AnalyzeProject(context.Background(), ".")
	if err != nil {
		log.Printf("Failed to analyze project: %v\n", err)
		return
	}

	// Print basic project information
	fmt.Printf("Project Name: %s\n", result.Name)
	fmt.Printf("Project Path: %s\n", result.Path)
	fmt.Printf("Analyzed At: %s\n\n", result.AnalyzedAt)

	// Print imports
	fmt.Println("Project Dependencies:")
	for _, imp := range result.Imports {
		if !strings.Contains(imp, "internal") && !strings.HasPrefix(imp, ".") {
			fmt.Printf("  - %s\n", imp)
		}
	}
}

func analyzeInterfaces(analyzer *readgo.DefaultAnalyzer) {
	pkgPath := "github.com/iamlongalong/readgo"
	interfaces := []string{"CodeAnalyzer", "SourceReader", "Validator"}

	for _, name := range interfaces {
		fmt.Printf("\nAnalyzing interface: %s\n", name)
		fmt.Println(strings.Repeat("-", 40))

		iface, err := analyzer.FindInterface(context.Background(), pkgPath, name)
		if err != nil {
			log.Printf("Failed to find interface %s: %v\n", name, err)
			continue
		}

		fmt.Printf("Name: %s\n", iface.Name)
		fmt.Printf("Package: %s\n", iface.Package)
		fmt.Printf("Type: %s\n", iface.Type)
		fmt.Printf("Exported: %v\n", iface.IsExported)
	}
}

func analyzeImplementations(analyzer *readgo.DefaultAnalyzer) {
	// Analyze DefaultAnalyzer implementation
	analyzeType(analyzer, "DefaultAnalyzer")

	// Analyze other types
	types := []string{"TreeOptions", "ReadOptions", "FileTreeNode", "TypeInfo", "FunctionInfo", "AnalysisResult"}
	for _, name := range types {
		analyzeType(analyzer, name)
	}
}

func analyzeType(analyzer *readgo.DefaultAnalyzer, typeName string) {
	fmt.Printf("\nAnalyzing type: %s\n", typeName)
	fmt.Println(strings.Repeat("-", 40))

	typeInfo, err := analyzer.FindType(context.Background(), "github.com/iamlongalong/readgo", typeName)
	if err != nil {
		log.Printf("Failed to find type %s: %v\n", typeName, err)
		return
	}

	fmt.Printf("Name: %s\n", typeInfo.Name)
	fmt.Printf("Package: %s\n", typeInfo.Package)
	fmt.Printf("Type: %s\n", typeInfo.Type)
	fmt.Printf("Exported: %v\n", typeInfo.IsExported)
}
