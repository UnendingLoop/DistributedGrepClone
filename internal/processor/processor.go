// Package processor operates with input - stdIn or file(s) - and sends result to output
package processor

import (
	"bufio"
	"fmt"
	"grepClone/internal/matcher"
	"grepClone/internal/output"
	"grepClone/internal/parser"
	"os"
)

func ProcessInput(fileName string) error {
	var scanner *bufio.Scanner
	var file *os.File

	// инициализация источника строк
	switch fileName {
	case "":
		scanner = bufio.NewScanner(os.Stdin)
	default:
		// проверяем открывается ли файл, и не папка ли это
		info, err := os.Stat(fileName)
		if err != nil { // если ошибка при открытии - проброс ошибки выше
			return fmt.Errorf("error opening file %q: %q", fileName, err)
		}
		if info.IsDir() { // если это папка - проброс ошибки выше
			return fmt.Errorf("specified source filename %q is a directory", fileName)
		}

		// открываем файл для чтения
		file, err = os.Open(fileName)
		if err != nil {
			return fmt.Errorf("couldn't open file %q: %q", fileName, err)
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	}

	// обработка при активном флаге "-с"
	if parser.SP.CountFound {
		countLinesToOutput(scanner, fileName)
		return nil
	}

	// стандартная обработка + поддержка контекста + реверс результата
	return stdProcessWithCtx(scanner, fileName)
}

func countLinesToOutput(scanner *bufio.Scanner, fileName string) {
	counter := 0
	for scanner.Scan() {
		if matcher.FindMatch(scanner.Text()) {
			counter++
		}
	}

	switch {
	case parser.SP.PrintFileName:
		fmt.Printf("%s: %d\n", fileName, counter)
	default:
		fmt.Println(counter)
	}
}

func stdProcessWithCtx(source *bufio.Scanner, fileName string) error {
	line := ""
	lineN := 1
	beforeBuf := make([]string, 0, parser.SP.CtxBefore)
	isCtxZone := false
	lastPrintedN := 0
	isMatch := false
	isPrinted := make(map[int]struct{})
	afterCount := 0

	for source.Scan() {
		line = source.Text()
		isMatch = matcher.FindMatch(line)
		if parser.SP.InvertResult { //-v
			isMatch = !isMatch
		}

		if isMatch {
			if !isCtxZone {
				// разбираемся с BEFORE и вставляем разделитель если надо
				j := lineN - len(beforeBuf)
				if j-lastPrintedN > 1 && lastPrintedN != 0 {
					output.PrintLine("--", "", 0)
				}
				for i := range beforeBuf {
					if _, ok := isPrinted[j]; !ok {
						output.PrintLine(beforeBuf[i], fileName, j)
						isPrinted[j] = struct{}{}
						lastPrintedN = j
					}
					j++
				}
			}

			// обработка самой isMatch-строки
			if _, ok := isPrinted[lineN]; !ok {
				output.PrintLine(line, fileName, lineN)
				lastPrintedN = lineN
				isPrinted[lineN] = struct{}{}
				if parser.SP.CtxAfter > 0 {
					isCtxZone = true
				}
				afterCount = parser.SP.CtxAfter
			}
			lineN++
			continue
		}

		// разбираемся с AFTER
		if afterCount > 0 {
			if _, ok := isPrinted[lineN]; !ok {
				output.PrintLine(line, fileName, lineN)
				lastPrintedN = lineN
				isPrinted[lineN] = struct{}{}
			}
			afterCount--
			if afterCount == 0 {
				isCtxZone = false
			}
		}

		// актуализируем beforeBuf
		if parser.SP.CtxBefore > 0 {
			if len(beforeBuf) == parser.SP.CtxBefore {
				beforeBuf = beforeBuf[1:] // pop front
			}
			beforeBuf = append(beforeBuf, line)
		}

		lineN++
	}

	return source.Err()
}
