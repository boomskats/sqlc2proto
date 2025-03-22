package parser

import (
	"os"
)

// Helper function to write a test file
func writeTestFile(path, content string) error {
	return writeFile(path, []byte(content))
}

// Helper function to write bytes to a file
func writeFile(path string, data []byte) error {
	f, err := createFile(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// Helper function to create a file
func createFile(path string) (*os.File, error) {
	return os.Create(path)
}
