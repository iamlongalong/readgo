package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/iamlongalong/readgo"
)

func main() {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "readgo-example")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	files := map[string]string{
		"valid.go": `package example

import "fmt"

// HelloWorld prints a greeting message
func HelloWorld() {
	fmt.Println("Hello, World!")
}`,
		"invalid.go": `package example

import "fmt"

// This file has syntax errors
func InvalidFunction() {
	fmt.Println("Missing closing parenthesis"
}`,
	}

	// Write test files
	for name, content := range files {
		if err := os.WriteFile(tmpDir+"/"+name, []byte(content), 0644); err != nil {
			log.Fatalf("Failed to write file %s: %v", name, err)
		}
	}

	// Create a new validator
	validator := readgo.NewValidator(tmpDir)

	// Validate individual files
	fmt.Println("Validating individual files:")
	for name := range files {
		fmt.Printf("\nValidating %s:\n", name)
		result, err := validator.ValidateFile(context.Background(), name)
		if err != nil {
			log.Printf("Error validating %s: %v", name, err)
			continue
		}

		if len(result.Errors) > 0 {
			fmt.Println("Errors:")
			for _, err := range result.Errors {
				fmt.Printf("  - %s\n", err)
			}
		} else {
			fmt.Println("No errors found")
		}

		if len(result.Warnings) > 0 {
			fmt.Println("Warnings:")
			for _, warn := range result.Warnings {
				fmt.Printf("  - %s: %s\n", warn.Type, warn.Message)
			}
		}
	}

	// Validate the entire project
	fmt.Println("\nValidating entire project:")
	result, err := validator.ValidateProject(context.Background())
	if err != nil {
		log.Fatalf("Failed to validate project: %v", err)
	}

	if len(result.Errors) > 0 {
		fmt.Println("Project errors:")
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
	} else {
		fmt.Println("No project-level errors found")
	}

	if len(result.Warnings) > 0 {
		fmt.Println("Project warnings:")
		for _, warn := range result.Warnings {
			fmt.Printf("  - %s: %s\n", warn.Type, warn.Message)
		}
	}
}
