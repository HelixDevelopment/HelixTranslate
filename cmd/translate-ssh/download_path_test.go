package main

import (
	"path/filepath"
	"testing"
)

// TestLocalDownloadPath_OutputEPUBHonorsConfiguredOutputFile is the RED-first
// reproduction (§11.4.115) of a real, user-impacting bug:
//
// The remote worker writes the translated EPUB to RemoteDir/<base(OutputFile)>
// (see step4ConvertToEPUB) and appends that remote path to FilesCreated.
// step5DownloadFiles then downloads EVERY created file to
// filepath.Join(filepath.Dir(InputFile), filepath.Base(remoteFile)).
//
// Consequence: when the operator passes -output pointing at a directory that
// differs from the input file's directory (or a different basename via -output),
// the translated EPUB is delivered to inputDir/<base> instead of the requested
// config.OutputFile. printFinalReport then claims "Output file: <OutputFile>"
// at a path where no file exists. The end user's requested artifact is never
// produced at the location they asked for.
//
// The fix: the EPUB output must be downloaded to exactly config.OutputFile.
func TestLocalDownloadPath_OutputEPUBHonorsConfiguredOutputFile(t *testing.T) {
	cfg := &Config{
		InputFile:  "/home/user/books/novel.fb2",
		OutputFile: "/srv/exports/novel_sr.epub", // different dir AND name
		RemoteDir:  "/tmp/translate-ssh",
	}

	// The remote EPUB path as produced by step4ConvertToEPUB.
	remoteEPUB := filepath.Join(cfg.RemoteDir, filepath.Base(cfg.OutputFile)) // /tmp/translate-ssh/novel_sr.epub

	got := localDownloadPath(remoteEPUB, cfg)

	if got != cfg.OutputFile {
		t.Fatalf("EPUB output must land at the operator-requested OutputFile.\n  remote:   %s\n  want:     %s\n  got:      %s",
			remoteEPUB, cfg.OutputFile, got)
	}
}

// TestLocalDownloadPath_MarkdownSidecarsGoToInputDir asserts the non-EPUB
// sidecar files (the _original.md / _translated.md the report lists relative to
// the input directory) keep landing next to the input file.
func TestLocalDownloadPath_MarkdownSidecarsGoToInputDir(t *testing.T) {
	cfg := &Config{
		InputFile:  "/home/user/books/novel.fb2",
		OutputFile: "/srv/exports/novel_sr.epub",
		RemoteDir:  "/tmp/translate-ssh",
	}

	remoteMD := filepath.Join(cfg.RemoteDir, "novel_translated.md")
	want := filepath.Join(filepath.Dir(cfg.InputFile), "novel_translated.md") // /home/user/books/novel_translated.md

	got := localDownloadPath(remoteMD, cfg)
	if got != want {
		t.Fatalf("markdown sidecar must land in the input directory.\n  remote: %s\n  want:   %s\n  got:    %s",
			remoteMD, want, got)
	}
}
