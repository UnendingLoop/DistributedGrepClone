package reader

import (
	"bufio"
	"fmt"
	"os"
)

func ReadInput(fileName string) ([]string, error) {
	switch fileName {
	case "":
		return readStdIn()
	default:
		return readFile(fileName)
	}
}

func readStdIn() ([]string, error) {
	result := make([]string, 0)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}
	return result, scanner.Err()
}

func readFile(fileName string) ([]string, error) {
	// проверяем открывается ли файл
	info, err := os.Stat(fileName)
	if err != nil {
		return nil, fmt.Errorf("error opening file %q: %q", fileName, err)
	}
	// проверяем не папка ли это
	if info.IsDir() {
		return nil, fmt.Errorf("specified source filename %q is a directory", fileName)
	}

	// открываем файл для чтения
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("couldn't open file %q: %q", fileName, err)
	}
	defer file.Close()

	// читаем и возвращаем результат
	result := make([]string, 0)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}
	return result, scanner.Err()
}
