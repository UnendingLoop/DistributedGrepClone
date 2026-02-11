// Package parser puts os.Args into SearchParam structure and validates it for any issues
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

func InitAppMode() (*model.AppInit, error) {
	var appInit model.AppInit
	flagParser := flag.NewFlagSet("DistributedGrepClone", flag.ExitOnError)
	mode := flagParser.String("mode", "", "specify mode of the app: 'master' or 'slave'")

	// парсим аргументы
	if err := flagParser.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	appInit.Mode = model.AppMode(*mode)

	// проверяем режим
	switch appInit.Mode {
	case model.ModeMaster:
		if err := initMasterParam(&appInit); err != nil {
			return nil, err
		}
		if len(*appInit.Slaves) == 0 {
			return nil, errors.New("at least one --node must be provided running in 'slave'-mode")
		}
	case model.ModeSlave:
		if err := initSlaveParam(&appInit); err != nil {
			return nil, err
		}
	default:
	}

	return &appInit, nil
}

func initSlaveParam(ai *model.AppInit) error {
	flagParser := flag.NewFlagSet("DistributedGrepClone", flag.ExitOnError)
	addr := flagParser.String("address", "", fmt.Sprintf("specify slave-node address (NB: %q is already use by master-node)", model.DefaultMasterAddress))

	if err := flagParser.Parse(os.Args[1:]); err != nil {
		return err
	}

	switch *addr {
	case "":
		return errors.New("empty slave-node address")
	case model.DefaultMasterAddress:
		return errors.New("specified slave-node address not available")
	default:
		ai.Address = *addr
		return nil
	}
}

func initMasterParam(ai *model.AppInit) error {
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

	q := flagParser.Int("quorum", 0, "set slave-nodes N for quorum")
	flagParser.Var(ai.Slaves, "node", "set slave-node address")

	if err := flagParser.Parse(os.Args[1:]); err != nil {
		return err
	}

	ai.SearchParam = model.GrepParam{
		CtxAfter:     *a,
		CtxBefore:    *b,
		CtxCircle:    *c,
		CountFound:   *d,
		IgnoreCase:   *e,
		InvertResult: *f,
		ExactMatch:   *g,
		EnumLine:     *h,
	}
	ai.Address = model.DefaultMasterAddress
	ai.Quorum = *q
	if ai.Quorum <= 0 {
		return errors.New("incorrect quorum N provided")
	}

	// Выравниваем значения контекста A и B по значению C
	setABCvaluesByPriority(&ai.SearchParam)

	// Приводим паттерн к нижнему регистру если стоят флаги 'F' и 'i'
	if ai.SearchParam.IgnoreCase && ai.SearchParam.ExactMatch {
		ai.SearchParam.Pattern = strings.ToLower(ai.SearchParam.Pattern)
	}

	// Разбираемся с паттерном и входом
	switch len(flagParser.Args()) {
	case 0:
		return errors.New("pattern not specified!\nUsage: DistributedGrepClone [flags] pattern [file...]")
	case 1:
		ai.SearchParam.Pattern = flagParser.Args()[0]
	default:
		ai.SearchParam.Pattern = flagParser.Args()[0]
		ai.SearchParam.Source = flagParser.Args()[1:]
	}

	// ставим флаг чтобы печатать имя файла перед каждой строкой/суммой строк, если файлов несколько
	if len(ai.SearchParam.Source) > 1 {
		ai.SearchParam.PrintFileName = true
	}

	// сразу првоеряем корректность регулярки, если флаг F неактивен
	if !ai.SearchParam.ExactMatch {
		var err error
		pattern := ai.SearchParam.Pattern
		if ai.SearchParam.IgnoreCase {
			pattern = "(?i)" + pattern
		}
		ai.SearchParam.RegexpPattern, err = regexp.Compile(pattern)
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
