package cmd

import (
	"log"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
	"github.com/UnendingLoop/DistributedGrepClone/internal/parser"
)

func main() {
	// инициализировать параметры запуска - режим и прочее:
	appParam, err := parser.InitAppMode()
	if err != nil {
		log.Printf("Failed to initialize the app: %q", err.Error())
		return
	}

	switch appParam.Mode {
	case model.ModeMaster: // вызов мастера
	case model.ModeSlave: // вызов слейва
	default:
		log.Printf("Failed to launch the app: unknown mode %q specified.\nExiting the app...", appParam.Mode)
		return
	}
}
