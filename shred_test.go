package main

import (
	"bytes"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestShredDeletesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "example.txt")

	err := os.WriteFile(path, []byte("Testing Go Shred"), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = Shred(path)
	if err != nil {
		t.Fatalf("Shred returned an unexpected error: %v", err)
	}

	_, err = os.Stat(path)

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be deleted, Stat returned: %v", err)
	}
}

func TestShredEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "example.txt")

	err := os.WriteFile(path, []byte{}, 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = Shred(path)
	if err != nil {
		t.Fatalf("Shred returned an unexpected error: %v", err)
	}

	_, err = os.Stat(path)

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be deleted, Stat returned: %v", err)
	}
}

func TestShredMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "example.txt")

	err := Shred(path)

	if err == nil {
		t.Fatalf("expected an error for a missing file, but got nil")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file-not-exist error, got: %v", err)
	}
}

func TestShredLargeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.bin")

	size := 100*1024 + 7
	data := make([]byte, size)
	_, err := rand.Read(data)
	if err != nil {
		t.Fatalf("failed to generate random data: %v", err)
	}

	err = os.WriteFile(path, data, 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = Shred(path)
	if err != nil {
		t.Fatalf("Shred returned an unexpected error: %v", err)
	}

	_, err = os.Stat(path)

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be deleted, Stat returned: %v", err)
	}
}

func TestShredDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subdirectory")

	err := os.Mkdir(path, 0700)
	if err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	err = Shred(path)
	if err == nil {
		t.Fatal("expected an error when shredding a directory")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected directory to still exist: %v", err)
	}

	if !info.IsDir() {
		t.Fatalf("expected path to remain a directory")
	}
}

func TestShredSymbolicLink(t *testing.T) {
	dir := t.TempDir()

	originalPath := filepath.Join(dir, "original.txt")
	linkPath := filepath.Join(dir, "link.txt")

	originalData := []byte("This content must remain unchanged")

	err := os.WriteFile(originalPath, originalData, 0600)
	if err != nil {
		t.Fatalf("failed to create original file: %v", err)
	}

	err = os.Symlink(originalPath, linkPath)
	if err != nil {
		t.Fatalf("failed to create symbolic link %v", err)
	}

	err = Shred(linkPath)
	if err == nil {
		t.Fatal("expected an error when shredding a symbolic link")
	}

	actualData, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("failed to read original file: %v", err)
	}

	if !bytes.Equal(originalData, actualData) {
		t.Fatalf("file contents changed: expected %q, got %q", originalData, actualData)
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("expected link to still exist: %v", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected path to remain a symbolic link")
	}
}

type recordingFile struct {
	*os.File
	seekPositions []int64
	passSizes     []int
	snapshots     [][]byte
	written       int
}

func (f *recordingFile) Seek(offset int64, whence int) (int64, error) {
	position, err := f.File.Seek(offset, whence)

	if err == nil {
		f.seekPositions = append(f.seekPositions, position)
	}

	return position, err
}

func (f *recordingFile) Write(data []byte) (int, error) {
	n, err := f.File.Write(data)

	f.written += n

	return n, err
}

func (f *recordingFile) Sync() error {
	if err := f.File.Sync(); err != nil {
		return err
	}

	snapshot, err := os.ReadFile(f.File.Name())
	if err != nil {
		return err
	}

	f.snapshots = append(f.snapshots, snapshot)
	f.passSizes = append(f.passSizes, f.written)

	f.written = 0

	return nil
}

func TestOverwriteThreePasses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overwrite.bin")

	err := os.WriteFile(path, []byte("ABCD"), 0600)
	if err != nil {
		t.Fatalf("failed to create test file %v", err)
	}

	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("failed to open test file: %v", err)
	}

	defer file.Close()

	recorded := &recordingFile{
		File: file,
	}

	random := bytes.NewReader([]byte("1234567890XY"))

	err = overwriteFile(recorded, 4, random)
	if err != nil {
		t.Fatalf("overwrite failed: %v", err)
	}

	if len(recorded.snapshots) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(recorded.snapshots))
	}

	if len(recorded.seekPositions) != 3 {
		t.Fatalf("expected 3 positions, got %d", len(recorded.seekPositions))
	}

	if len(recorded.passSizes) != 3 {
		t.Fatalf("expected 3 passes, got %d", len(recorded.passSizes))
	}

	expectedSnapshots := [][]byte{
		[]byte("1234"),
		[]byte("5678"),
		[]byte("90XY"),
	}

	for i, expected := range expectedSnapshots {
		if !bytes.Equal(recorded.snapshots[i], expected) {
			t.Fatalf("pass %d: expected %q, got %q",
				i+1, expected, recorded.snapshots[i])
		}

		if recorded.seekPositions[i] != 0 {
			t.Fatalf("pass %d: expected seek positions 0 , got %d",
				i+1, recorded.seekPositions[i])
		}

		if recorded.passSizes[i] != 4 {
			t.Fatalf("pass %d: expected 4 bytes written, got %d",
				i+1, recorded.passSizes[i])
		}
	}
	if random.Len() != 0 {
		t.Fatalf("expected all random bytes to be consumed, got %d re,aomomg", random.Len())
	}
}
