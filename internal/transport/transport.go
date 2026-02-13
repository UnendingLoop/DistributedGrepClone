package transport

import (
	"net/http"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
	"github.com/UnendingLoop/DistributedGrepClone/internal/processor"
	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/ginext"
)

func NewSlaveServer(addr string) *http.Server {
	engine := ginext.New("debug")
	engine.GET("/ping", HealthCheck)
	engine.POST("/task", ReceiveTask)

	return &http.Server{
		Addr:    addr,
		Handler: engine,
	}
}

func HealthCheck(ctx *ginext.Context) {
	ctx.Status(200)
}

func ReceiveTask(ctx *ginext.Context) {
	var task *model.SlaveTask

	if err := ctx.ShouldBind(task); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"failed to parse task from body: ": err.Error()})
	}

	res := processor.ProcessInput(ctx.Request.Context(), task)

	ctx.JSON(http.StatusOK, res)
}
