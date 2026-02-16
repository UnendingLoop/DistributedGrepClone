// Package model contains data structure for storing initially provided flags, input-source values and DTO
package model

import (
	"context"
	"fmt"
)

type AppMode string

const (
	ModeMaster = AppMode("master")
	ModeSlave  = AppMode("slave")
)

type AppInit struct {
	Mode        AppMode
	Address     string
	Slaves      NodesList
	Quorum      int
	SearchParam GrepParam
}

// NodesList - для чтения списка slave-nodes в виде слайса из OS.args
type NodesList []string

func (n *NodesList) String() string {
	return fmt.Sprint(*n)
}

func (n *NodesList) Set(value string) error { // в Set сразу избавляемся от дубликатов slave-node адресов
	var found bool
	for _, v := range *n {
		if v == value {
			found = true
		}
	}
	if !found || value != "" {
		*n = append(*n, value)
	}
	return nil
}

// GrepParam - хранит в себе все возможные флаги и параметры запуска grep
type GrepParam struct {
	CtxAfter      int      `json:"ctx_after"`                  // A n — вывести N строк после каждой найденной строки
	CtxBefore     int      `json:"ctx_before"`                 // B n — вывести N строк до каждой найденной строки
	CtxCircle     int      `json:"-"`                          // C N — вывести N строк контекста вокруг найденной строки (включает и до, и после; эквивалентно -A N -B N)
	CountFound    bool     `json:"count_found"`                // c — выводить только число совпавших с шаблоном строк,  -n/-A/-B/-C при этом игнорируются
	IgnoreCase    bool     `json:"ignore_case"`                // i — игнорировать регистр
	InvertResult  bool     `json:"invert_result"`              // v — инвертировать фильтр: выводить строки, не содержащие шаблон
	ExactMatch    bool     `json:"exact_match"`                // F — выполнять точное совпадение подстроки - вето на регулярку
	EnumLine      bool     `json:"enum_line"`                  // n — выводить номер строки перед каждой найденной строкой.
	Source        []string `json:"-"`                          // Имя/имена файлов для чтения данных
	Pattern       string   `json:"pattern" binding:"required"` // raw Regexp или строка для поиска
	PrintFileName bool     `json:"print_filename"`             // used to print filename prefix if there are >1 files to process
}

type MasterTask struct {
	Task      TaskDTO
	CTX       context.Context
	CancelCTX context.CancelFunc
}
type TaskDTO struct {
	TaskID   string    `json:"tid" binding:"required"`
	GP       GrepParam `json:"grep_param" binding:"required"`
	Input    []string  `json:"input" binding:"required"`
	FileName string    `json:"file_name,omitempty"`
}

type SlaveTask struct {
	TaskID   string    `json:"tid" binding:"required"`
	GP       GrepParam `json:"grep_param" binding:"required"`
	Input    []string  `json:"input" binding:"required"`
	FileName string    `json:"file_name,omitempty"`
}
type SlaveResult struct {
	TaskID   string   `json:"tid" binding:"required"`
	HashSumm uint64   `json:"hash" binding:"required"`
	Output   []string `json:"output" binding:"required"`
}
