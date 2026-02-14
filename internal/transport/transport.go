// Package transport provides a new server-entity(by ginext) for slave-mode operability with handlers to serve endpoints
package transport

import (
	"log"
	"net/http"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
	"github.com/UnendingLoop/DistributedGrepClone/internal/processor"
	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/ginext"
)

func NewSlaveServer(addr string) *http.Server {
	engine := ginext.New("release")
	engine.GET("/ping", HealthCheck)
	engine.POST("/task", ReceiveTask)

	return &http.Server{
		Addr:    addr,
		Handler: engine,
	}
}

func HealthCheck(ctx *ginext.Context) {
	log.Println("Received a healthcheck request!")
	ctx.Status(200)
}

func ReceiveTask(ctx *ginext.Context) {
	var task model.SlaveTask

	if err := ctx.ShouldBindJSON(&task); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"failed to parse task from body: ": err.Error()})
	}

	log.Printf("Received task: %q", task.TaskID)
	log.Printf("Input array is: %v", task.Input)

	res := processor.ProcessInput(ctx.Request.Context(), &task)
	log.Printf("Calculated result: %v", res)

	ctx.JSON(http.StatusOK, res)
}
