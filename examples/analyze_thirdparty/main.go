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

	// Analyze a third-party package
	analyzePackage(analyzer, "golang.org/x/tools/go/packages")
	analyzePackage(analyzer, "github.com/stretchr/testify/assert")
}

func analyzePackage(analyzer *readgo.DefaultAnalyzer, pkgPath string) {
	fmt.Printf("\nAnalyzing package: %s\n", pkgPath)
	fmt.Println(strings.Repeat("-", 80))

	// Analyze the package
	result, err := analyzer.AnalyzePackage(context.Background(), pkgPath)
	if err != nil {
		log.Printf("Failed to analyze package %s: %v\n", pkgPath, err)
		return
	}

	// Print package information
	fmt.Printf("Package: %s\n", result.Name)
	fmt.Printf("Path: %s\n", result.Path)

	// Print types
	fmt.Println("\nTypes:")
	for _, t := range result.Types {
		if t.IsExported {
			fmt.Printf("  - %s.%s: %s\n", t.Package, t.Name, t.Type)
		}
	}

	// Print functions
	fmt.Println("\nFunctions:")
	for _, f := range result.Functions {
		if f.IsExported {
			fmt.Printf("  - %s.%s\n", f.Package, f.Name)
		}
	}

	// Print imports
	fmt.Println("\nImports:")
	for _, imp := range result.Imports {
		fmt.Printf("  - %s\n", imp)
	}

	// Try to find some specific types
	findType(analyzer, pkgPath, "Package")
	findInterface(analyzer, pkgPath, "Interface")
}

func findType(analyzer *readgo.DefaultAnalyzer, pkgPath, typeName string) {
	fmt.Printf("\nLooking for type %s in %s\n", typeName, pkgPath)
	typeInfo, err := analyzer.FindType(context.Background(), pkgPath, typeName)
	if err != nil {
		fmt.Printf("Type not found: %v\n", err)
		return
	}

	fmt.Printf("Found type:\n")
	fmt.Printf("  Name: %s\n", typeInfo.Name)
	fmt.Printf("  Package: %s\n", typeInfo.Package)
	fmt.Printf("  Type: %s\n", typeInfo.Type)
	fmt.Printf("  Exported: %v\n", typeInfo.IsExported)
}

func findInterface(analyzer *readgo.DefaultAnalyzer, pkgPath, interfaceName string) {
	fmt.Printf("\nLooking for interface %s in %s\n", interfaceName, pkgPath)
	interfaceInfo, err := analyzer.FindInterface(context.Background(), pkgPath, interfaceName)
	if err != nil {
		fmt.Printf("Interface not found: %v\n", err)
		return
	}

	fmt.Printf("Found interface:\n")
	fmt.Printf("  Name: %s\n", interfaceInfo.Name)
	fmt.Printf("  Package: %s\n", interfaceInfo.Package)
	fmt.Printf("  Type: %s\n", interfaceInfo.Type)
	fmt.Printf("  Exported: %v\n", interfaceInfo.IsExported)
}
