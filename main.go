package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
)

func main() {
	file, err := os.CreateTemp(".", "practice-*.txt")
	if err != nil {
		fmt.Println("Failed to create file:", err)
		return
	}

	defer file.Close()

	bytesWritten, err := file.WriteString("Testing Go file operations\n")
	if err != nil {
		fmt.Println("Failed to write to file:", err)
		return
	}

	bytesWrittenToo, err := file.WriteString("New line of text\n")
	if err != nil {
		fmt.Println("Failed to write to file:", err)
		return
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		fmt.Println("Failed to seek:", err)
		return
	}

	_, err = file.WriteString("XX")
	if err != nil {
		fmt.Println("Failed to overwrite:", err)
		return
	}

	fmt.Printf("First write: %d bytes written\n", bytesWritten)
	fmt.Printf("Second write: %d bytes written\n", bytesWrittenToo)

	info, err := file.Stat()
	if err != nil {
		fmt.Println("Failed to get file information:", err)
		return
	}

	fileSize := info.Size()
	fmt.Printf("File size: %d bytes\n", fileSize)

	data := make([]byte, 8)

	_, err = rand.Read(data)
	if err != nil {
		fmt.Println("Failed to generate random bytes:", err)
		return
	}

	fmt.Printf("Random bytes %v\n", data)

	data2 := make([]byte, 8)

	for pass := 1; pass <= 3; pass++ {
		fmt.Printf("Starting pass %d\n", pass)

		passWritten := int64(0)

		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			fmt.Println("Failed to seek:", err)
			return
		}

		remaining := fileSize

		for remaining > 0 {

			_, err = rand.Read(data2)
			if err != nil {
				fmt.Println("Failed to generate random bytes:", err)
				return
			}

			chunk := data2

			if remaining < int64(len(chunk)) {
				chunk = chunk[:int(remaining)]
			}

			bytesWritten, err = file.Write(chunk)
			if err != nil {
				fmt.Println("Failed to overwrite:", err)
				return
			}

			if bytesWritten != len(chunk) {
				fmt.Println("Incomplete write")
				return
			}

			passWritten += int64(bytesWritten)
			remaining -= int64(bytesWritten)

		}
		err = file.Sync()
		if err != nil {
			fmt.Println("Failed to synchronise file:", err)
			return
		}

		if passWritten != fileSize {
			fmt.Printf("Incomplete pass: wrote %d of %d bytes\n", passWritten, fileSize)
			return
		}

		fmt.Printf("Completed pass %d: %d bytes overwritten\n", pass, passWritten)
	}
}
