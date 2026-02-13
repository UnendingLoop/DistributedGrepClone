// Package parser puts os.Args into SearchParam structure and validates it for any business-issues
package parser

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
)

var ctxPriority = map[string]int{}

func InitAppMode(osArgs []string) (*model.AppInit, error) {
	var appInit model.AppInit
	flagParser := flag.NewFlagSet("DistributedGrepClone", flag.ExitOnError)
	mode := flagParser.String("mode", "", "specify mode of the app: 'master' or 'slave'")
	a := flagParser.Int("A", 0, "show N lines after target line")
	b := flagParser.Int("B", 0, "show N lines before target line")
	c := flagParser.Int("C", 0, "show N lines before and after target line(same as '-A N' and '-B N')")
	d := flagParser.Bool("c", false, "show only total number of matching lines(A/B/C/n are ignored)")
	e := flagParser.Bool("i", false, "all input lines will be lower-cased for search as well as the pattern itself")
	f := flagParser.Bool("v", false, "search only lines that DON'T match the specified pattern")
	g := flagParser.Bool("F", false, "specified pattern will be used strictly as a string, not regexp")
	h := flagParser.Bool("n", false, "enumerates output lines according to their order in input")
	addr := flagParser.String("addr", "", "specify slave-node address")

	q := flagParser.Int("quorum", -1, "set slave-nodes N for quorum")
	flagParser.Var(&appInit.Slaves, "node", "set slave-node address")

	// парсим аргументы
	if err := flagParser.Parse(osArgs); err != nil {
		return nil, err
	}

	appInit.Mode = model.AppMode(*mode)

	// проверяем режим
	switch appInit.Mode {
	case model.ModeMaster:
		appInit.SearchParam = model.GrepParam{
			CtxAfter:     *a,
			CtxBefore:    *b,
			CtxCircle:    *c,
			CountFound:   *d,
			IgnoreCase:   *e,
			InvertResult: *f,
			ExactMatch:   *g,
			EnumLine:     *h,
		}
		appInit.Quorum = *q

		if err := initMasterParam(&appInit, flagParser.Args()); err != nil {
			return nil, err
		}
		if len(appInit.Slaves) == 0 {
			return nil, errors.New("at least one --node must be provided running in 'slave'-mode")
		}
	case model.ModeSlave:
		if *addr == "" {
			return nil, errors.New("empty slave-node address")
		}
		appInit.Address = *addr
	}

	return &appInit, nil
}

func initMasterParam(ai *model.AppInit, noNameArgs []string) error {
	preprocessArgs()

	if len(ai.Slaves) == 0 {
		return errors.New("no slave-nodes specified")
	}

	switch {
	case ai.Quorum == 0:
		return errors.New("incorrect quorum N provided")
	case ai.Quorum < 0: // если ничего не передано - ставим значение по умолчанию: половина числа slave-nodes + 1
		ai.Quorum = (len(ai.Slaves) / 2) + 1
	case ai.Quorum <= len(ai.Slaves):
		return errors.New("slave-nodes N cannot be less than --quorum value")
	}

	// Выравниваем значения контекста A и B по значению C
	setABCvaluesByPriority(&ai.SearchParam)

	// Приводим паттерн к нижнему регистру если стоят флаги 'F' и 'i'
	if ai.SearchParam.IgnoreCase && ai.SearchParam.ExactMatch {
		ai.SearchParam.Pattern = strings.ToLower(ai.SearchParam.Pattern)
	}

	// Разбираемся с паттерном и входом
	switch len(noNameArgs) {
	case 0:
		return errors.New("pattern not specified!\nUsage: DistributedGrepClone [flags] pattern [file(s)...]")
	case 1:
		ai.SearchParam.Pattern = noNameArgs[0]
	default:
		ai.SearchParam.Pattern = noNameArgs[0]
		ai.SearchParam.Source = noNameArgs[1:]
	}

	// ставим флаг чтобы печатать имя файла перед каждой строкой/суммой строк, если файлов несколько
	if len(ai.SearchParam.Source) > 1 {
		ai.SearchParam.PrintFileName = true
	}

	// сразу првоеряем корректность регулярки, если флаг F неактивен
	if !ai.SearchParam.ExactMatch {
		var err error
		if ai.SearchParam.IgnoreCase {
			ai.SearchParam.Pattern = "(?i)" + ai.SearchParam.Pattern
		}
		_, err = regexp.Compile(ai.SearchParam.Pattern)
		if err != nil {
			return fmt.Errorf("incorrect regexp provided: %q", err.Error())
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

func setABCvaluesByPriority(SP *model.GrepParam) {
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
