package main

import (
	"fmt"
	"archive/zip"
)

func main() {
	// First create a test EPUB
	fmt.Println("Creating test EPUB...")
	// This would use createSimpleEPUB from the test
	
	// Check format detection
	//detector := format.NewDetector()
	//fmt.Println("Detector created")
	
	// List files in an existing EPUB if it exists
	r, err := zip.OpenReader("/Users/milosvasic/Projects/Translate/test.epub")
	if err != nil {
		fmt.Printf("Error opening test EPUB: %v\n", err)
		return
	}
	defer r.Close()
	
	fmt.Println("Files in EPUB:")
	for _, f := range r.File {
		fmt.Printf("  %s\n", f.Name)
	}
}