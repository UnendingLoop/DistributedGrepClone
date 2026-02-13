// Package processor operates with input task and sends result back to transport-layer
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
		result.Output = countMatchingLines(ctx, task.Input, task.FileName, &task.GP)
	default:
		result.Output = getMatchingLines(ctx, task.Input, task.FileName, &task.GP)
	}

	// считаем общий хеш
	result.HashSumm = hasher(ctx, result.Output)

	return &result
}

func countMatchingLines(ctx context.Context, input []string, fileName string, gp *model.GrepParam) []string { // не нужно ли переделать чтобы возвращалась только строка, а не слайс?
	res := ""
	counter := 0
	for _, v := range input {
		select {
		case <-ctx.Done():
			return nil
		default:
			if findMatch(gp, v) {
				counter++
			}
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

func getMatchingLines(ctx context.Context, input []string, fileName string, gp *model.GrepParam) []string {
	var result []string
	lineN := 1
	beforeBuf := make([]string, 0, gp.CtxBefore)
	isCtxZone := false
	lastPrintedN := 0
	isMatch := false
	isPrinted := make(map[int]struct{})
	afterCount := 0

	for _, line := range input {
		select {
		case <-ctx.Done():
			return nil
		default: // всю дефолтную ветку можно вынести в отдельную функцию внутри этой функции для читабельности
			isMatch = findMatch(gp, line)
			if gp.InvertResult { //-v
				isMatch = !isMatch
			}

			if isMatch {
				if !isCtxZone {
					// разбираемся с BEFORE и вставляем разделитель если надо
					j := lineN - len(beforeBuf)
					if j-lastPrintedN > 1 && lastPrintedN != 0 {
						result = append(result, normalizeLine(gp, "--", "", 0))
					}
					for i := range beforeBuf {
						if _, ok := isPrinted[j]; !ok {
							result = append(result, normalizeLine(gp, beforeBuf[i], fileName, j))
							isPrinted[j] = struct{}{}
							lastPrintedN = j
						}
						j++
					}
				}

				// обработка самой isMatch-строки
				if _, ok := isPrinted[lineN]; !ok {
					result = append(result, normalizeLine(gp, line, fileName, lineN))
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
					result = append(result, normalizeLine(gp, line, fileName, lineN))
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
	}

	return result
}

// учесть что нужно делать префикс имени файла + нумерация строк
func normalizeLine(SP *model.GrepParam, line, fileName string, n int) string {
	switch {
	case fileName == "":
		return line
	case SP.PrintFileName && SP.EnumLine:
		return fmt.Sprintf("%s: %d: %s", fileName, n, line)
	case SP.EnumLine:
		return fmt.Sprintf("%d: %s", n, line)
	case SP.PrintFileName:
		return fmt.Sprintf("%s: %s", fileName, line)
	default:
		return line
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

func hasher(ctx context.Context, input []string) uint64 {
	hs := xxhash.New()
	for _, s := range input {
		select {
		case <-ctx.Done():
			return 0
		default:
			_, _ = hs.WriteString(s)
		}
	}
	return hs.Sum64()
}
