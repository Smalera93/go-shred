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
		t.Fatalf("expected an error when shredding a directory")
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
		t.Fatalf("file centents changed: expected %q, got %q", originalData, actualData)
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("expected link to still exist: %v", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected path to remain a symbolic link")
	}
}
