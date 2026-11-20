package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
)

// overwriteTarget defines the file operations required by overwriteFile
type overwriteTarget interface {
	io.WriteSeeker
	Sync() error
}

// Shred overwrites a regular file three times with random data,
// synchronises each pass and removes the file after successful completion.
func Shred(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("failed to inspect file: %w", err)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", path)
	}

	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	needsClose := true

	defer func() {
		if needsClose {
			_ = file.Close()
		}
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to inspect opened file: %w", err)
	}

	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("opened file is not a regular file")
	}

	if !os.SameFile(info, fileInfo) {
		return fmt.Errorf("file changed between inspection and opening")
	}

	fileSize := fileInfo.Size()

	if err := overwriteFile(file, fileSize, rand.Reader); err != nil {
		return fmt.Errorf("failed to overwrite file: %w", err)
	}

	closeErr := file.Close()
	needsClose = false

	if closeErr != nil {
		return fmt.Errorf("failed to close file: %w", closeErr)
	}

	currentInfo, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("failed to inspect file before removal: %w", err)
	}

	if !os.SameFile(fileInfo, currentInfo) {
		return fmt.Errorf("file changed before removal")
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	return nil
}

// overwriteFile performs three overwrite passes using the supplied
// data source and a fixed-size buffer to limit memory usage.
func overwriteFile(file overwriteTarget, fileSize int64, random io.Reader) error {

	data := make([]byte, 32*1024)

	for pass := 1; pass <= 3; pass++ {

		passWritten := int64(0)

		_, err := file.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("failed to seek: %w", err)
		}

		remaining := fileSize

		for remaining > 0 {
			chunk := data

			if remaining < int64(len(chunk)) {
				chunk = chunk[:int(remaining)]
			}

			_, err := io.ReadFull(random, chunk)
			if err != nil {
				return fmt.Errorf("failed to generate random bytes: %w", err)
			}

			bytesWritten, err := file.Write(chunk)
			if err != nil {
				return fmt.Errorf("failed to overwrite file: %w", err)
			}

			if bytesWritten != len(chunk) {
				return fmt.Errorf("incomplete write: wrote %d of %d bytes", bytesWritten, len(chunk))
			}

			passWritten += int64(bytesWritten)
			remaining -= int64(bytesWritten)

		}
		err = file.Sync()
		if err != nil {
			return fmt.Errorf("failed to synchronise file: %w", err)
		}

		if passWritten != fileSize {
			return fmt.Errorf("incomplete pass: wrote %d of %d bytes", passWritten, fileSize)
		}
	}

	return nil
}
