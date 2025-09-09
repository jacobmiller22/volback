package zip

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestCreateArchiveFromPath_Directory(t *testing.T) {

	r, err := CreateArchiveFromPath("./testdata/testdirectory")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read all of the contents and write this to a tempfile, use the `zip` command to validate it

	tmpDir := t.TempDir()
	tmpFilename := filepath.Join(tmpDir, "testdirectory.zip")

	fd, err := os.Create(tmpFilename)
	if err != nil {
		t.Fatalf("unexpected error creating zip disk archive: %v", err)
	}

	if _, err := io.Copy(fd, r); err != nil {
		t.Fatalf("unexpected error copying contents: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)

	// Run zip command to validate
	unzippedFilename := filepath.Join(tmpDir, "testdirectory.unzipped")
	cmd := exec.CommandContext(ctx, "unzip", tmpFilename, "-d", unzippedFilename)
	cmd.Start()
	cmd.Wait()
	cancel()

	if cmd.ProcessState.ExitCode() != 0 {
		t.Fatalf("returned error code %d, expected 0", cmd.ProcessState.ExitCode())
	}

	cmd = exec.Command("ls", tmpDir)
	output, _ := cmd.CombinedOutput()

	results := strings.Split(strings.Trim(string(output), " \n"), "\n")

	if len(results) != 2 {
		t.Errorf("Expected 2 results but got %d: %v", len(results), results)
	}

	if !slices.Contains(results, "testdirectory.unzipped") || !slices.Contains(results, "testdirectory.zip") {
		t.Errorf("expected testdirectory.unzipped and testdirectory.zip, got: %v", results)
	}
}

func TestCreateArchiveFromPath_Path(t *testing.T) {

	r, err := CreateArchiveFromPath("./testdata/testfile.txt")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read all of the contents and write this to a tempfile, use the `zip` command to validate it
	// tmpDir := "./testdata"
	tmpDir := t.TempDir()
	tmpFilename := filepath.Join(tmpDir, "testfile.zip")

	fd, err := os.Create(tmpFilename)
	if err != nil {
		t.Fatalf("unexpected error creating zip disk archive: %v", err)
	}

	if _, err := io.Copy(fd, r); err != nil {
		t.Fatalf("unexpected error copying contents: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)

	// Run unzip command to validate
	unzippedFilepath := filepath.Join(tmpDir, "testfile.unzipped")
	// Open the output file for writing the unzipped contents
	outFd, err := os.Create(unzippedFilepath)
	if err != nil {
		t.Fatalf("unexpected error creating unzipped file: %v", err)
	}
	defer outFd.Close()

	// Use unzip -p to write the file contents to stdout, then copy to our file
	cmd := exec.CommandContext(ctx, "unzip", "-p", tmpFilename)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("unexpected error getting stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("unexpected error starting unzip: %v", err)
	}

	if _, err := io.Copy(outFd, stdout); err != nil {
		t.Fatalf("unexpected error copying unzipped contents: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("unexpected error waiting for unzip: %v", err)
	}
	cancel()

	cmd = exec.Command("ls", tmpDir)
	output, _ := cmd.CombinedOutput()

	results := strings.Split(strings.Trim(string(output), " \n"), "\n")

	if len(results) != 2 {
		t.Errorf("Expected 2 results but got %d: %v", len(results), results)
	}

	if !slices.Contains(results, "testfile.zip") || !slices.Contains(results, "testfile.unzipped") {
		t.Errorf("expected ttestfile.zip and testfile.unzipped, got: %v", results)
	}

	// Read the unzipped file and validate its contents
	unzippedData, err := os.ReadFile(unzippedFilepath)
	if err != nil {
		t.Fatalf("unexpected error reading unzipped file: %v", err)
	}

	originalData, err := os.ReadFile("./testdata/testfile.txt")
	if err != nil {
		t.Fatalf("unexpected error reading original file: %v", err)
	}

	if diff := cmp.Diff(string(originalData), string(unzippedData)); diff != "" {
		t.Errorf("unzipped file contents do not match original (-want +got):\n%s", diff)
	}
}

func TestUnpackArchiveToPath_Path(t *testing.T) {

	// Open the test zip file
	zipFile, err := os.Open("./testdata/testfile.zip")
	if err != nil {
		t.Fatalf("unexpected error opening zip file: %v", err)
	}
	defer zipFile.Close()

	// Create a temporary directory for extraction
	tmpDir := t.TempDir()
	unpackPath := filepath.Join(tmpDir, "testfile.unzipped")

	// Call UnpackArchiveToPath to extract the archive
	if err := UnpackArchiveToPath(zipFile, unpackPath); err != nil {
		t.Fatalf("unexpected error unpacking archive: %v", err)
	}

	// List the directory and check that testfile.txt exists
	files, err := os.ReadDir(unpackPath)
	if err != nil {
		t.Fatalf("unexpected error reading unpacked directory: %v", err)
	}

	found := false
	for _, f := range files {
		if f.Name() == "testfile.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected testfile.txt to exist in unpacked directory, got: %v", files)
	}
}

func TestUnpackArchiveToPath_Directory(t *testing.T) {
	// Open the test zip file
	zipFile, err := os.Open("./testdata/testdirectory.zip")
	if err != nil {
		t.Fatalf("unexpected error opening zip file: %v", err)
	}
	defer zipFile.Close()

	// Create a temporary directory for extraction
	tmpDir := t.TempDir()
	unpackPath := filepath.Join(tmpDir, "testdirectory.unzipped")

	// Call UnpackArchiveToPath to extract the archive
	if err := UnpackArchiveToPath(zipFile, unpackPath); err != nil {
		t.Fatalf("unexpected error unpacking archive: %v", err)
	}

	unpackPath = filepath.Join(unpackPath, "testdirectory")

	// Validate the file structure:
	// unpackPath/archivepath/
	//   a/
	//     a.txt
	//   b.txt

	// Check for "a" directory and "b.txt" file
	entries, err := os.ReadDir(unpackPath)
	if err != nil {
		t.Fatalf("unexpected error reading unpacked directory: %v", err)
	}

	hasADir := false
	hasBTxt := false
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == "a" {
			hasADir = true
		}
		if !entry.IsDir() && entry.Name() == "b.txt" {
			hasBTxt = true
		}
	}
	if !hasADir {
		t.Errorf("expected directory 'a' in unpacked directory, got: %v", entries)
	}
	if !hasBTxt {
		t.Errorf("expected file 'b.txt' in unpacked directory, got: %v", entries)
	}

	// Check for "a.txt" inside "a" directory
	aDirPath := filepath.Join(unpackPath, "a")
	aEntries, err := os.ReadDir(aDirPath)
	if err != nil {
		t.Fatalf("unexpected error reading 'a' directory: %v", err)
	}
	hasATxt := false
	for _, entry := range aEntries {
		if !entry.IsDir() && entry.Name() == "a.txt" {
			hasATxt = true
			break
		}
	}
	if !hasATxt {
		t.Errorf("expected file 'a.txt' in directory 'a', got: %v", aEntries)
	}
}
