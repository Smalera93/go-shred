package main

import (
	"fmt"
	"os"
)

func main() {
	file, err := os.CreateTemp(".", "practice-*.txt")
	if err != nil {
		fmt.Println("Failed to create test file:", err)
		return
	}

	path := file.Name()

	_, err = file.WriteString("Testing Go file operations\nNew line of text\n")
	if err != nil {
		file.Close()
		fmt.Println("Failed to write test file:", err)
		return
	}

	if err := file.Close(); err != nil {
		fmt.Println("Failed to close test file:", err)
		return
	}

	fmt.Println("Created test file:", path)

	err = Shred(path)
	if err != nil {
		fmt.Println("Shared failed:", err)
		return
	}

	fmt.Println("Shred completed successfully.")

	_, err = os.Stat(path)

	if os.IsNotExist(err) {
		fmt.Println("Verified: test file no longer exists.")
	} else if err != nil {
		fmt.Println("Could not verify deletion:", err)
	} else {
		fmt.Println("Warning: test file still exists.")
	}
}
