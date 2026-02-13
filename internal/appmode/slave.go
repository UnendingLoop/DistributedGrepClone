package appmode

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
	"github.com/UnendingLoop/DistributedGrepClone/internal/transport"
)

func RunSlave(ctx context.Context, stop context.CancelFunc, ai *model.AppInit) {
	// получить экземпляр сервера
	srv := transport.NewSlaveServer(ai.Address)

	// запуск сервера
	go func() {
		log.Printf("Slave running on %s", srv.Addr)
		err := srv.ListenAndServe()
		if err != nil {
			switch {
			case errors.Is(err, http.ErrServerClosed):
				log.Println("Server gracefully stopping...")
			default:
				log.Printf("Server stopped: %v", err)
				stop()
			}
		}
	}()

	<-ctx.Done()

	// Закрытие всех соединений сервера
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Failed to shutdown slave-node %q correctly: %q", ai.Address, err.Error())
	} else {
		log.Printf("Slave-node %q server is closed.", ai.Address)
	}
}
