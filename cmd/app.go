package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/UnendingLoop/DistributedGrepClone/internal/appmode"
	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
	"github.com/UnendingLoop/DistributedGrepClone/internal/parser"
)

func main() {
	// инициализировать параметры запуска - режим и прочее:
	appParam, err := parser.InitAppMode(os.Args[1:])
	if err != nil {
		log.Printf("Failed to launch DistributedGrepClone: %q", err.Error())
		return
	}

	// готовим слушатель прерываний - контекст для всего приложения
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// запуск приложения в указанном режиме
	switch appParam.Mode {
	case model.ModeMaster:
		appmode.RunMaster(ctx, stop, appParam)
	case model.ModeSlave:
		appmode.RunSlave(ctx, stop, appParam)
	default:
		log.Printf("Failed to launch: unknown mode %q specified.\nExiting the app...", appParam.Mode)
		return
	}
}
