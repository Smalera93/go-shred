package main

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestFile(t *testing.T, data []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "overwrite.bin")

	err := os.WriteFile(path, data, 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	return path
}

func openTestFile(t *testing.T, data []byte) (string, *os.File) {
	t.Helper()

	path := createTestFile(t, data)

	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("failed to open test file: %v", err)
	}

	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Errorf("failed to close test file: %v", err)
		}
	})

	return path, file
}

func assertFileContents(t *testing.T, path string, expected []byte) {
	t.Helper()

	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	if !bytes.Equal(actual, expected) {
		t.Fatalf("expected file contents %q, got %q",
			expected, actual)
	}
}

func assertFileDeleted(t *testing.T, path string) {
	t.Helper()

	_, err := os.Lstat(path)

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be deleted, Lstat returned: %v", err)
	}
}

func TestShredDeletesFile(t *testing.T) {
	path := createTestFile(t, []byte("Testing Go Shred"))

	err := Shred(path)
	if err != nil {
		t.Fatalf("Shred returned an unexpected error: %v", err)
	}

	assertFileDeleted(t, path)
}

func TestShredEmptyFile(t *testing.T) {
	path := createTestFile(t, []byte{})

	err := Shred(path)
	if err != nil {
		t.Fatalf("Shred returned an unexpected error: %v", err)
	}

	assertFileDeleted(t, path)
}

func TestShredMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "example.txt")

	err := Shred(path)

	if err == nil {
		t.Fatal("expected an error for a missing file, but got nil")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file-not-exist error, got: %v", err)
	}
}

func TestShredLargeFile(t *testing.T) {
	size := 100*1024 + 7
	data := make([]byte, size)
	_, err := rand.Read(data)
	if err != nil {
		t.Fatalf("failed to generate random data: %v", err)
	}

	path := createTestFile(t, data)

	err = Shred(path)
	if err != nil {
		t.Fatalf("Shred returned an unexpected error: %v", err)
	}

	assertFileDeleted(t, path)
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
		t.Fatal("expected path to remain a directory")
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
		t.Fatalf("failed to create symbolic link: %v", err)
	}

	err = Shred(linkPath)
	if err == nil {
		t.Fatal("expected an error when shredding a symbolic link")
	}

	assertFileContents(t, originalPath, originalData)

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
	seekError     error
	writeError    error
	shortWrite    bool
	syncError     error
}

func (f *recordingFile) Seek(offset int64, whence int) (int64, error) {
	if f.seekError != nil {
		return 0, f.seekError
	}

	position, err := f.File.Seek(offset, whence)

	if err == nil {
		f.seekPositions = append(f.seekPositions, position)
	}

	return position, err
}

func (f *recordingFile) Write(data []byte) (int, error) {
	if f.writeError != nil {
		return 0, f.writeError
	}

	if f.shortWrite {
		data = data[:len(data)/2]
	}

	n, err := f.File.Write(data)

	f.written += n

	return n, err
}

func (f *recordingFile) Sync() error {
	if f.syncError != nil {
		return f.syncError
	}

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
	_, file := openTestFile(t, []byte("ABCD"))

	recorded := &recordingFile{
		File: file,
	}

	random := bytes.NewReader([]byte("1234567890XY"))

	err := overwriteFile(recorded, 4, random)
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
			t.Fatalf("pass %d: expected seek position 0, got %d",
				i+1, recorded.seekPositions[i])
		}

		if recorded.passSizes[i] != 4 {
			t.Fatalf("pass %d: expected 4 bytes written, got %d",
				i+1, recorded.passSizes[i])
		}
	}
	if random.Len() != 0 {
		t.Fatalf("expected all random bytes to be consumed, got %d remaining", random.Len())
	}
}

func TestOverwriteRandomFailure(t *testing.T) {
	path, file := openTestFile(t, []byte("ABCD"))

	recorded := &recordingFile{
		File: file,
	}

	random := bytes.NewReader([]byte("123456"))

	err := overwriteFile(recorded, 4, random)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected unexpected EOF error, got: %v", err)
	}

	if len(recorded.snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(recorded.snapshots))
	}

	if len(recorded.seekPositions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(recorded.seekPositions))
	}

	if len(recorded.passSizes) != 1 {
		t.Fatalf("expected 1 pass, got %d", len(recorded.passSizes))
	}

	for i, position := range recorded.seekPositions {
		if position != 0 {
			t.Fatalf("pass %d: expected seek position 0, got %d",
				i+1, position)
		}
	}

	if recorded.passSizes[0] != 4 {
		t.Fatalf("expected 4 bytes written, got %d",
			recorded.passSizes[0])
	}

	if recorded.written != 0 {
		t.Fatalf("expected no bytes written during failed pass, got %d",
			recorded.written)
	}

	expectedData := []byte("1234")

	if !bytes.Equal(recorded.snapshots[0], expectedData) {
		t.Fatalf("expected snapshot %q, got %q",
			expectedData, recorded.snapshots[0])
	}

	if random.Len() != 0 {
		t.Fatalf("expected all random bytes to be consumed, got %d remaining", random.Len())
	}

	assertFileContents(t, path, expectedData)
}

func TestOverwriteSeekFailure(t *testing.T) {
	path, file := openTestFile(t, []byte("ABCD"))

	expectedErr := errors.New("simulated seek failure")

	recorded := &recordingFile{
		File:      file,
		seekError: expectedErr,
	}

	random := bytes.NewReader([]byte("1234567890XY"))

	err := overwriteFile(recorded, 4, random)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected seek failure, got: %v", err)
	}

	if len(recorded.seekPositions) != 0 {
		t.Fatalf("expected 0 positions, got %d", len(recorded.seekPositions))
	}

	if len(recorded.snapshots) != 0 {
		t.Fatalf("expected 0 snapshots, got %d", len(recorded.snapshots))
	}

	if len(recorded.passSizes) != 0 {
		t.Fatalf("expected 0 completed passes, got %d", len(recorded.passSizes))
	}

	if recorded.written != 0 {
		t.Fatalf("expected no bytes written during failed pass, got %d", recorded.written)
	}

	if random.Len() != 12 {
		t.Fatalf("expected 12 unread bytes, got %d", random.Len())
	}

	assertFileContents(t, path, []byte("ABCD"))
}

func TestOverwriteWriteFailure(t *testing.T) {
	path, file := openTestFile(t, []byte("ABCD"))

	expectedErr := errors.New("simulated write failure")

	recorded := &recordingFile{
		File:       file,
		writeError: expectedErr,
	}

	random := bytes.NewReader([]byte("1234567890XY"))

	err := overwriteFile(recorded, 4, random)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected write failure, got: %v", err)
	}

	if len(recorded.seekPositions) != 1 {
		t.Fatalf("expected 1 successful seek, got %d", len(recorded.seekPositions))
	}

	if len(recorded.snapshots) != 0 {
		t.Fatalf("expected 0 snapshots, got %d", len(recorded.snapshots))
	}

	if len(recorded.passSizes) != 0 {
		t.Fatalf("expected 0 completed passes, got %d", len(recorded.passSizes))
	}

	if recorded.written != 0 {
		t.Fatalf("expected no bytes written during failed pass, got %d", recorded.written)
	}

	if random.Len() != 8 {
		t.Fatalf("expected 8 unread bytes, got %d", random.Len())
	}

	assertFileContents(t, path, []byte("ABCD"))
}

func TestOverwriteShortWrite(t *testing.T) {
	path, file := openTestFile(t, []byte("ABCD"))

	recorded := &recordingFile{
		File:       file,
		shortWrite: true,
	}

	random := bytes.NewReader([]byte("1234567890XY"))

	err := overwriteFile(recorded, 4, random)

	if err == nil {
		t.Fatal("expected an incomplete write error")
	}

	if !strings.Contains(err.Error(), "incomplete write") {
		t.Fatalf("expected incomplete write error, got: %v", err)
	}

	if len(recorded.seekPositions) != 1 {
		t.Fatalf("expected 1 successful seek, got %d", len(recorded.seekPositions))
	}

	if len(recorded.snapshots) != 0 {
		t.Fatalf("expected 0 snapshots, got %d", len(recorded.snapshots))
	}

	if len(recorded.passSizes) != 0 {
		t.Fatalf("expected 0 completed passes, got %d", len(recorded.passSizes))
	}

	if recorded.written != 2 {
		t.Fatalf("expected 2 bytes written, got %d", recorded.written)
	}

	if random.Len() != 8 {
		t.Fatalf("expected 8 unread bytes, got %d", random.Len())
	}

	assertFileContents(t, path, []byte("12CD"))
}

func TestOverwriteSyncFailure(t *testing.T) {
	path, file := openTestFile(t, []byte("ABCD"))

	expectedErr := errors.New("simulated sync failure")

	recorded := &recordingFile{
		File:      file,
		syncError: expectedErr,
	}

	random := bytes.NewReader([]byte("1234567890XY"))

	err := overwriteFile(recorded, 4, random)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected sync failure, got: %v", err)
	}

	if len(recorded.seekPositions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(recorded.seekPositions))
	}

	if len(recorded.snapshots) != 0 {
		t.Fatalf("expected 0 snapshots, got %d", len(recorded.snapshots))
	}

	if len(recorded.passSizes) != 0 {
		t.Fatalf("expected 0 completed passes, got %d", len(recorded.passSizes))
	}

	if recorded.written != 4 {
		t.Fatalf("expected 4 bytes written, got %d", recorded.written)
	}

	if random.Len() != 8 {
		t.Fatalf("expected 8 unread bytes, got %d", random.Len())
	}

	assertFileContents(t, path, []byte("1234"))
}
