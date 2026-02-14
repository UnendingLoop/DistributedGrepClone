// Package transport provides a new server-entity(by ginext) for slave-mode operability with handlers to serve endpoints
package transport

import (
	"context"
	"log"
	"net/http"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/ginext"
)

type GrepHandler struct {
	Proc TaskProcessor
}

type TaskProcessor interface {
	ProcessInput(ctx context.Context, task *model.SlaveTask) *model.SlaveResult
}

func NewSlaveServer(addr string, p TaskProcessor) *http.Server {
	h := GrepHandler{
		Proc: p,
	}

	engine := ginext.New("release")
	engine.GET("/ping", h.HealthCheck)
	engine.POST("/task", h.ReceiveTask)

	return &http.Server{
		Addr:    addr,
		Handler: engine,
	}
}

func (gh GrepHandler) HealthCheck(ctx *ginext.Context) {
	log.Println("Received a healthcheck request!")
	ctx.Status(200)
}

func (gh GrepHandler) ReceiveTask(ctx *ginext.Context) {
	var task model.SlaveTask

	if err := ctx.ShouldBindJSON(&task); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"failed to parse task from body: ": err.Error()})
		return
	}

	res := gh.Proc.ProcessInput(ctx.Request.Context(), &task)

	ctx.JSON(http.StatusOK, res)
}
