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

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		fmt.Println("Failed to seek:", err)
		return
	}

	data2 := make([]byte, 8)
	remaining := fileSize

	for remaining > 0 {

		_, err = rand.Read(data2)
		if err != nil {
			fmt.Println("Failed to generate random bytes:", err)
			return
		}

		if remaining < int64(len(data2)) {
			data2 = data2[:int(remaining)]
		}

		bytesWritten, err = file.Write(data2)
		if err != nil {
			fmt.Println("Failed to overwrite:", err)
			return
		}
		remaining -= int64(bytesWritten)
	}
}
