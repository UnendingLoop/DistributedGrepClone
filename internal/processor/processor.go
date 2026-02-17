// Package processor operates with input task and sends result back to transport-layer
package processor

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
	"github.com/cespare/xxhash/v2"
)

type Processor struct{}

func (p Processor) ProcessInput(ctx context.Context, task *model.SlaveTask) *model.SlaveResult {
	result := model.SlaveResult{
		TaskID: task.TaskID,
	}

	// считаем метчи или выводим метчи
	switch task.GP.CountFound {
	case true:
		res := countMatchingLines(ctx, task.Input, task.FileName, &task.GP)
		if res == "" {
			result.Output = []string{}
		} else {
			result.Output = []string{res}
		}

	default:
		result.Output = getMatchingLines(ctx, task.Input, task.FileName, &task.GP)
	}

	// считаем общий хеш
	result.HashSumm = hasher(ctx, result.Output)

	return &result
}

func countMatchingLines(ctx context.Context, input []string, fileName string, gp *model.GrepParam) string {
	result := ""
	counter := 0
	for _, v := range input {
		select {
		case <-ctx.Done():
			return ""
		default:
			match, err := findMatch(gp, v)
			if err != nil {
				log.Printf("problem with pattern %q: %v", gp.Pattern, err)
				return result
			}
			if match {
				counter++
			}
		}
	}

	switch {
	case gp.PrintFileName:
		result = fmt.Sprintf("%s:%d", fileName, counter)
	default:
		result = fmt.Sprintln(counter)
	}
	return result
}

func getMatchingLines(ctx context.Context, input []string, fileName string, gp *model.GrepParam) []string {
	result := []string{}
	lineN := 1
	beforeBuf := make([]string, 0, gp.CtxBefore)
	isCtxZone := false
	lastPrintedN := 0
	isMatch := false
	isPrinted := make(map[int]struct{})
	afterCount := 0

	withCTX := true
	if gp.CtxAfter == 0 && gp.CtxBefore == 0 {
		withCTX = false
	}

	var err error

	for _, line := range input {
		select {
		case <-ctx.Done():
			return []string{}
		default: // всю дефолтную ветку можно вынести в отдельную функцию внутри этой функции для читабельности
			isMatch, err = findMatch(gp, line)
			if err != nil {
				log.Printf("problem with pattern %q: %v", gp.Pattern, err)
				return result
			}
			if gp.InvertResult { //-v
				isMatch = !isMatch
			}

			switch withCTX {
			case true:
				if isMatch {
					if !isCtxZone {
						// разбираемся с BEFORE и вставляем разделитель если надо
						j := lineN - len(beforeBuf)
						if j-lastPrintedN > 1 && lastPrintedN != 0 {
							result = append(result, "--")
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
			default:
				if isMatch {
					result = append(result, normalizeLine(gp, line, fileName, lineN))
				}
			}

			lineN++
		}
	}

	return result
}

// учесть что нужно делать префикс имени файла + нумерация строк
func normalizeLine(SP *model.GrepParam, line, fileName string, n int) string {
	switch {
	case SP.PrintFileName && SP.EnumLine:
		return fmt.Sprintf("%s:%d:%s", fileName, n, line)
	case SP.EnumLine:
		return fmt.Sprintf("%d:%s", n, line)
	case SP.PrintFileName:
		return fmt.Sprintf("%s:%s", fileName, line)
	default:
		return line
	}
}

func findMatch(SP *model.GrepParam, line string) (bool, error) {
	if SP.IgnoreCase { //-i
		line = strings.ToLower(line)
	}

	switch {
	case SP.ExactMatch: //-F
		return strings.Contains(line, SP.Pattern), nil
	default:
		pattern, err := regexp.Compile(SP.Pattern)
		if err != nil {
			return false, err
		}
		return pattern.MatchString(line), nil
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
