package main

import (
	"fmt"
	"os"
	"archive/zip"
	"strings"
	"io"
	"digital.vasic.translator/pkg/format"
)

func main() {
	// Create a simple test EPUB using the createEPUB approach from MarkdownToEPUBConverter
	epubPath := "/tmp/test_epub.epub"
	
	// Create EPUB file
	epubFile, err := os.Create(epubPath)
	if err != nil {
		fmt.Printf("Error creating EPUB: %v\n", err)
		return
	}
	defer epubFile.Close()
	
	// Create ZIP writer
	zipWriter := zip.NewWriter(epubFile)
	
	// Write mimetype (must be uncompressed)
	mimeWriter, err := zipWriter.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // No compression
	})
	if err != nil {
		fmt.Printf("Error creating mimetype: %v\n", err)
		return
	}
	mimeWriter.Write([]byte("application/epub+zip"))
	
	// Write container.xml
	containerWriter, err := zipWriter.Create("META-INF/container.xml")
	if err != nil {
		fmt.Printf("Error creating container: %v\n", err)
		return
	}
	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	containerWriter.Write([]byte(containerXML))
	
	// Write content.opf
	contentWriter, err := zipWriter.Create("OEBPS/content.opf")
	if err != nil {
		fmt.Printf("Error creating content.opf: %v\n", err)
		return
	}
	contentOPF := `<?xml version="1.0" encoding="UTF-8"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf">
  <metadata>
    <dc:title>Test Book</dc:title>
    <dc:creator>Test Author</dc:creator>
  </metadata>
  <manifest>
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
  </spine>
</package>`
	contentWriter.Write([]byte(contentOPF))
	
	// Write chapter
	chapterWriter, err := zipWriter.Create("OEBPS/chapter1.xhtml")
	if err != nil {
		fmt.Printf("Error creating chapter: %v\n", err)
		return
	}
	chapterXHTML := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
    <title>Chapter 1</title>
</head>
<body>
    <h1>Chapter 1</h1>
    <p>This is test content.</p>
</body>
</html>`
	chapterWriter.Write([]byte(chapterXHTML))
	
	// Close ZIP writer
	zipWriter.Close()
	
	// Now check the detected format
	detector := format.NewDetector()
	detectedFormat, err := detector.DetectFile(epubPath)
	if err != nil {
		fmt.Printf("Error detecting format: %v\n", err)
		return
	}
	
	fmt.Printf("Detected format: %s\n", detectedFormat)
	
	// Debug: check what mimetype the file has
	r2, err := zip.OpenReader(epubPath)
	if err != nil {
		fmt.Printf("Error opening EPUB: %v\n", err)
		return
	}
	defer r2.Close()
	
	for _, f := range r2.File {
		if f.Name == "mimetype" {
			rc, err := f.Open()
			if err != nil {
				fmt.Printf("Error opening mimetype: %v\n", err)
				return
			}
			defer rc.Close()
			
			data, err := io.ReadAll(rc)
			if err != nil {
				fmt.Printf("Error reading mimetype: %v\n", err)
				return
			}
			
			fmt.Printf("Mimetype content: %q\n", string(data))
			break
		}
	}
	
	// List files in the EPUB and check indicators
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		fmt.Printf("Error opening EPUB: %v\n", err)
		return
	}
	defer r.Close()
	
	fmt.Println("Files in EPUB:")
	for _, f := range r.File {
		fmt.Printf("  %s\n", f.Name)
	}
	
	// Debug: check what the isAZW3File function looks for
	generalIndicators := []string{
		"mimetype",
		"OEBPS",
		"META-INF",
	}
	
	azw3Specific := []string{
		"kindle:embed",
		"amzn-eastock",
		"kindle-fonts",
		"kindle:enclosure",
		"kindle:meta",
	}
	
	hasGeneral := 0
	hasSpecific := false
	
	for _, f := range r.File {
		fmt.Printf("Checking file: %s\n", f.Name)
		// Check for general indicators
		for _, indicator := range generalIndicators {
			if f.Name == indicator || strings.Contains(f.Name, indicator) {
				hasGeneral++
				fmt.Printf("  Found general indicator: %s (total: %d)\n", indicator, hasGeneral)
				break
			}
		}
		
		// Check for AZW3-specific indicators
		for _, specific := range azw3Specific {
			if strings.Contains(f.Name, specific) {
				hasSpecific = true
				fmt.Printf("  Found AZW3-specific indicator: %s\n", specific)
				break
			}
		}
	}
	
	fmt.Printf("General indicators: %d, AZW3 specific: %v\n", hasGeneral, hasSpecific)
	fmt.Printf("Would be detected as AZW3: %v\n", hasGeneral >= 2 && hasSpecific)
}