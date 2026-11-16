package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
)

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

	data := make([]byte, 32*1034)
	fileSize := info.Size()

	for pass := 1; pass <= 3; pass++ {
		fmt.Printf("Starting pass %d\n", pass)

		passWritten := int64(0)

		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("failed to seek: %w", err)
		}

		remaining := fileSize

		for remaining > 0 {

			_, err = rand.Read(data)
			if err != nil {
				return fmt.Errorf("failed to generate random bytes: %w", err)
			}

			chunk := data

			if remaining < int64(len(chunk)) {
				chunk = chunk[:int(remaining)]
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

		fmt.Printf("completed pass %d: %d bytes overwritten\n", pass, passWritten)
	}

	closeErr := file.Close()
	needsClose = false

	if closeErr != nil {
		return fmt.Errorf("failed to close file: %w", closeErr)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove file: %w", closeErr)
	}

	return nil
}
