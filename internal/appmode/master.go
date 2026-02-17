// Package appmode provides 2 methods to work in preliminarily defined modes 'master' and 'slave'
package appmode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
	"github.com/UnendingLoop/DistributedGrepClone/internal/qaggr"
	"github.com/UnendingLoop/DistributedGrepClone/internal/reader"
	"github.com/docker/distribution/uuid"
)

func RunMaster(ctx context.Context, stop context.CancelFunc, ai *model.AppInit) {
	defer stop()
	// прочитать все инпут-строки и преобразовать в задания
	tasks, err := readInputConvertToTasks(ctx, ai.SearchParam.Source, ai.SearchParam)
	if err != nil {
		log.Printf("Failed to read input: %v", err)
		return
	}

	// проверить пингом, что хотя бы минимальное кол-во slave-nodes доступны
	if err := checkSlavesHealth(ctx, ai.Slaves, ai.Quorum); err != nil {
		log.Printf("Failed to start grepping: %v", err)
		return
	}

	// асинхронно:
	// - отправить всем зарегистрированным слейвам задания
	// - получить результаты
	result, err := processTasks(ctx, ai.Slaves, tasks, ai.Quorum)
	if err != nil {
		log.Printf("Failed to grep: %v", err)
		return
	}

	// печатаем результат
	for _, v := range result {
		for _, line := range v {
			fmt.Println(line)
		}
	}
}

func checkSlavesHealth(ctx context.Context, slavesAddr []string, quorumN int) error {
	wg := sync.WaitGroup{}
	var goodSlaves atomic.Int64
	rCtx, cancel := context.WithTimeout(ctx, 5*time.Second) // 5 секунд на обнаружение всех slave-nodes
	defer cancel()

	client := &http.Client{}

	for _, v := range slavesAddr {
		if !strings.Contains(v, "http://") {
			v = "http://" + v
		}
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			req, err := http.NewRequestWithContext(rCtx, "GET", addr+"/ping", nil)
			if err != nil {
				return
			}

			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				goodSlaves.Add(1)
			}
		}(v)
	}

	wg.Wait()
	res := goodSlaves.Load()
	if res < int64(quorumN) { // если кол-во OK меньше quorumN+1, возвращаем ошибку
		return fmt.Errorf("only %d slave-nodes are OK to continue, while quorum should be %d", res, quorumN)
	}

	return nil
}

func readInputConvertToTasks(ctx context.Context, src []string, gp model.GrepParam) ([]*model.MasterTask, error) {
	var tasks []*model.MasterTask
	var input []string
	var err error

	// преобразовать вход в задания
	switch len(src) {
	case 0: // читаем вход из stdIn
		input, err = reader.ReadInput(os.Stdin, "")
		if err != nil {
			return nil, err
		}
		tCTX, cancel := context.WithCancel(ctx)
		tasks = append(tasks, &model.MasterTask{
			Task: model.TaskDTO{
				TaskID: uuid.Generate().String(),
				GP:     gp,
				Input:  input,
			},
			CTX:       tCTX,
			CancelCTX: cancel,
		})
	case 1: // читаем из единственного файла
		input, err = reader.ReadInput(os.Stdin, src[0])
		if err != nil {
			return nil, err
		}

		tCTX, cancel := context.WithCancel(ctx)
		tasks = append(tasks, &model.MasterTask{
			Task: model.TaskDTO{
				TaskID:   uuid.Generate().String(),
				GP:       gp,
				Input:    input,
				FileName: src[0],
			},
			CTX:       tCTX,
			CancelCTX: cancel,
		})
	default: // итерируемся по списку файлов
		for _, fname := range src {
			input, err = reader.ReadInput(os.Stdin, fname)
			if err != nil {
				return nil, err
			}

			tCTX, cancel := context.WithTimeout(ctx, 1*time.Minute)
			tasks = append(tasks, &model.MasterTask{
				Task: model.TaskDTO{
					TaskID:   uuid.Generate().String(),
					GP:       gp,
					Input:    input,
					FileName: fname,
				},
				CTX:       tCTX,
				CancelCTX: cancel,
			})
		}
	}

	return tasks, nil
}

func processTasks(ctx context.Context, nodes []string, tasks []*model.MasterTask, quorumN int) ([][]string, error) {
	resCollect := make(chan model.SlaveResult)

	// итерируемся по заданиям(их может быть несколько, если на вход подано несколько файлов)
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	wg := sync.WaitGroup{}
	for i := range tasks {
		task := tasks[i]
		// итерируемся по всем slave-node адресам и отправляем задания
		for _, nodeAddr := range nodes {
			wg.Add(1)
			go sendTaskToNode(task.CTX, &wg, &client, nodeAddr, task, resCollect)
		}
	}

	// в отдельной горутине отслеживаем завершение работы пишущих в канал горутин sendTaskToNode()
	go func() {
		wg.Wait()
		close(resCollect)
	}()

	// запускаем сборщика результатов c таймаутом в 1 минуту на сбор
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	return qaggr.CollectAggregateResults(ctx, resCollect, tasks, quorumN)
}

func sendTaskToNode(ctx context.Context, wg *sync.WaitGroup, client *http.Client, na string, task *model.MasterTask, ch chan<- model.SlaveResult) {
	defer wg.Done()

	if !strings.Contains(na, "http://") {
		na = "http://" + na
	}

	// сразу маршалим задание на отправку
	raw, err := json.Marshal(task.Task)
	if err != nil {
		log.Printf("failed to MARSHAL task: %q", err.Error())
		return
	}

	body := bytes.NewReader(raw)

	req, err := http.NewRequestWithContext(task.CTX, "POST", na+"/task", body)
	if err != nil {
		log.Printf("failed to GENERATE request to slave-node %q: %q", na, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to SEND task to slave-node: %q", err.Error())
		return
	}

	defer resp.Body.Close()

	var result model.SlaveResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("failed to UNMARSHAL result from slave-node %q: %q", na, err.Error())
		return
	}

	select {
	case ch <- result:
	case <-ctx.Done():
		return
	}
}
