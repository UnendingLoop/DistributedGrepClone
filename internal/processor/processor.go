// Package processor operates with input - stdIn or file(s) - and sends result to output
package processor

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
	"github.com/cespare/xxhash/v2"
)

func ProcessInput(ctx context.Context, task *model.SlaveTask) *model.SlaveResult {
	result := model.SlaveResult{
		TaskID: task.TaskID,
	}

	// считаем метчи или выводим метчи
	switch task.GP.CountFound {
	case true:
		result.Output = countMatchingLines(task.Input, task.FileName, &task.GP)
	default:
		result.Output = getMatchingLines(task.Input, task.FileName, &task.GP)
	}

	// считаем общий хеш
	result.HashSumm = hasher(result.Output)

	return &result
}

func countMatchingLines(input []string, fileName string, gp *model.GrepParam) []string {
	res := ""
	counter := 0
	for _, v := range input {
		if findMatch(gp, v) {
			counter++
		}
	}

	switch {
	case gp.PrintFileName:
		res = fmt.Sprintf("%s: %d\n", fileName, counter)
	default:
		res = fmt.Sprintln(counter)
	}
	return []string{res}
}

func getMatchingLines(input []string, fileName string, gp *model.GrepParam) []string {
	lineN := 1
	beforeBuf := make([]string, 0, gp.CtxBefore)
	isCtxZone := false
	lastPrintedN := 0
	isMatch := false
	isPrinted := make(map[int]struct{})
	afterCount := 0

	for _, line := range input {
		isMatch = findMatch(gp, line)
		if gp.InvertResult { //-v
			isMatch = !isMatch
		}

		if isMatch {
			if !isCtxZone {
				// разбираемся с BEFORE и вставляем разделитель если надо
				j := lineN - len(beforeBuf)
				if j-lastPrintedN > 1 && lastPrintedN != 0 {
					printLine(gp, "--", "", 0)
				}
				for i := range beforeBuf {
					if _, ok := isPrinted[j]; !ok {
						printLine(gp, beforeBuf[i], fileName, j)
						isPrinted[j] = struct{}{}
						lastPrintedN = j
					}
					j++
				}
			}

			// обработка самой isMatch-строки
			if _, ok := isPrinted[lineN]; !ok {
				printLine(gp, line, fileName, lineN)
				lastPrintedN = lineN
				isPrinted[lineN] = struct{}{}
				if gp.CtxAfter > 0 {
					isCtxZone = true
				}
				afterCount = gp.CtxAfter
			}
			lineN++
			continue
		}

		// разбираемся с AFTER
		if afterCount > 0 {
			if _, ok := isPrinted[lineN]; !ok {
				printLine(gp, line, fileName, lineN)
				lastPrintedN = lineN
				isPrinted[lineN] = struct{}{}
			}
			afterCount--
			if afterCount == 0 {
				isCtxZone = false
			}
		}

		// актуализируем beforeBuf
		if gp.CtxBefore > 0 {
			if len(beforeBuf) == gp.CtxBefore {
				beforeBuf = beforeBuf[1:] // pop front
			}
			beforeBuf = append(beforeBuf, line)
		}

		lineN++
	}

	return nil
}

// учесть что нужно делать префикс имени файла если файлов >1  + нумерация строк
func printLine(SP *model.GrepParam, line, fileName string, n int) {
	switch {
	case fileName == "":
		fmt.Println(line)
	case SP.PrintFileName && SP.EnumLine:
		fmt.Printf("%s: %d: %s\n", fileName, n, line)
	case SP.EnumLine:
		fmt.Printf("%d: %s\n", n, line)
	case SP.PrintFileName:
		fmt.Printf("%s: %s\n", fileName, line)
	default:
		fmt.Println(line)
	}
}

func findMatch(SP *model.GrepParam, line string) bool {
	if SP.IgnoreCase { //-i
		line = strings.ToLower(line)
	}

	switch {
	case SP.ExactMatch: //-F
		return strings.Contains(line, SP.Pattern)
	default:
		pattern, _ := regexp.Compile(SP.Pattern)
		return pattern.MatchString(line)
	}
}

func hasher(input []string) uint64 {
	hs := xxhash.New()
	for _, s := range input {
		_, _ = hs.WriteString(s)
	}
	return hs.Sum64()
}
