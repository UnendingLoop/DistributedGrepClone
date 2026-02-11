// Package parser puts os.Args into SearchParam structure and validates it for any issues
package parser

import (
	"errors"
	"flag"
	"os"
	"regexp"
	"strings"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
)

var (
	SP          model.SearchParam
	ctxPriority = map[string]int{}
)

func InitSearchParam() error {
	preprocessArgs()

	flagParser := flag.NewFlagSet("grepClone", flag.ExitOnError)

	a := flagParser.Int("A", 0, "show N lines after target line")
	b := flagParser.Int("B", 0, "show N lines before target line")
	c := flagParser.Int("C", 0, "show N lines before and after target line(same as '-A N' and '-B N')")
	d := flagParser.Bool("c", false, "show only total number of matching lines(A/B/C/n are ignored)")
	e := flagParser.Bool("i", false, "all input lines will be lower-cased for search as well as the pattern itself")
	f := flagParser.Bool("v", false, "search only lines that DON'T match the specified pattern")
	g := flagParser.Bool("F", false, "specified pattern will be used strictly as a string, not regexp")
	h := flagParser.Bool("n", false, "enumerates output lines according to their order in input")

	if err := flagParser.Parse(os.Args[1:]); err != nil {
		return err
	}

	SP = model.SearchParam{
		CtxAfter:     *a,
		CtxBefore:    *b,
		CtxCircle:    *c,
		CountFound:   *d,
		IgnoreCase:   *e,
		InvertResult: *f,
		ExactMatch:   *g,
		EnumLine:     *h,
	}

	// Выравниваем значения контекста A и B по значению C
	setABCvaluesByPriority()

	// Приводим паттерн к нижнему регистру если стоят флаги 'F' и 'i'
	if SP.IgnoreCase && SP.ExactMatch {
		SP.Pattern = strings.ToLower(SP.Pattern)
	}

	// Разбираемся с паттерном и входом
	switch len(flagParser.Args()) {
	case 0:
		return errors.New("pattern not specified!\nUsage: grepClone [flags] pattern [file...]")
	case 1:
		SP.Pattern = flagParser.Args()[0]
	default:
		SP.Pattern = flagParser.Args()[0]
		SP.Source = flagParser.Args()[1:]
	}

	// ставим флаг чтобы печатать имя файла перед каждой строкой/суммой строк, если файлов несколько
	if len(SP.Source) > 1 {
		SP.PrintFileName = true
	}

	// сразу првоеряем корректность регулярки, если флаг F неактивен
	if !SP.ExactMatch {
		var err error
		pattern := SP.Pattern
		if SP.IgnoreCase {
			pattern = "(?i)" + pattern
		}
		SP.RegexpPattern, err = regexp.Compile(pattern)
		if err != nil {
			return err
		}
	}
	return nil
}

func preprocessArgs() {
	counter := 1
	for _, arg := range os.Args {
		// определяем приоритет флагов выставления контекста по порядку их встречаемости в аргументах
		switch {
		case strings.Contains(arg, "-A"):
			ctxPriority["A"] = counter
			counter++
		case strings.Contains(arg, "-B"):
			ctxPriority["B"] = counter
			counter++
		case strings.Contains(arg, "-C"):
			ctxPriority["C"] = counter
			counter++
		}
	}
}

func setABCvaluesByPriority() {
	a := ctxPriority["A"]
	b := ctxPriority["B"]
	c, ok := ctxPriority["C"]

	if ok {
		switch {
		case c > a && c > b:
			SP.CtxAfter = SP.CtxCircle
			SP.CtxBefore = SP.CtxCircle
		case c > b:
			SP.CtxBefore = SP.CtxCircle
		case c > a:
			SP.CtxAfter = SP.CtxCircle
		}
	}
}
