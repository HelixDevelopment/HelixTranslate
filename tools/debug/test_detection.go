package main

import (
	"digital.vasic.translator/pkg/format"
	"fmt"
	"os"
	"reflect"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test_detection.go <epub_file>")
		os.Exit(1)
	}

	filename := os.Args[1]

	detector := format.NewDetector()

	// Test detection
	detectedFormat, err := detector.DetectFile(filename)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Detected format: %s\n", detectedFormat.String())

	// Use reflection to call isAZW3File method
	detectorValue := reflect.ValueOf(detector)
	method := detectorValue.MethodByName("isAZW3File")
	if !method.IsValid() {
		fmt.Println("isAZW3File method not found")
		return
	}

	result := method.Call([]reflect.Value{reflect.ValueOf(filename)})
	fmt.Printf("isAZW3File: %v\n", result[0].Bool())
}
