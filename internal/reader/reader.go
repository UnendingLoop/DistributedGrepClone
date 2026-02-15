// Package reader provides means of reading input specified at launch:
// - stdIn if no filenames is specified,
// - local file.
package reader

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func ReadInput(stdIn io.Reader, fileName string) ([]string, error) {
	switch fileName {
	case "":
		return readStdIn(stdIn)
	default:
		return readFile(fileName)
	}
}

func readStdIn(stdIn io.Reader) ([]string, error) {
	result := make([]string, 0)
	scanner := bufio.NewScanner(stdIn)
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}
	return result, scanner.Err()
}

func readFile(fileName string) ([]string, error) {
	// проверяем открывается ли файл
	info, err := os.Stat(fileName)
	if err != nil {
		return nil, fmt.Errorf("error opening file %q: %v", fileName, err)
	}
	// проверяем не папка ли это
	if info.IsDir() {
		return nil, fmt.Errorf("specified source filename %q is a directory", fileName)
	}

	// открываем файл для чтения
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("couldn't open file %q: %v", fileName, err)
	}
	defer file.Close()

	// читаем и возвращаем результат
	result := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}
	return result, scanner.Err()
}
