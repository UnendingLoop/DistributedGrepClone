// Package model contains data structure for storing initially provided flags and input-source values
package model

import "regexp"

type SearchParam struct {
	CtxAfter      int      // A n — вывести N строк после найденной строки
	CtxBefore     int      // B n — вывести N строк до каждой найденной строки
	CtxCircle     int      // C N — вывести N строк контекста вокруг найденной строки (включает и до, и после; эквивалентно -A N -B N)
	CountFound    bool     // c — выводить только число совпавших с шаблоном строк,  -n/-A/-B/-C при этом игнорируются
	IgnoreCase    bool     // i — игнорировать регистр
	InvertResult  bool     // v — инвертировать фильтр: выводить строки, не содержащие шаблон
	ExactMatch    bool     // F — выполнять точное совпадение подстроки - вето на регулярку
	EnumLine      bool     // n — выводить номер строки перед каждой найденной строкой.
	Source        []string // Имя/имена файлов для чтения данных
	Pattern       string   // Regexp или строка для поиска
	PrintFileName bool     // used to print filename prefix if there are >1 files to process
	RegexpPattern *regexp.Regexp
}
