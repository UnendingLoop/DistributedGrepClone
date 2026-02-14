// Package appmode provides 2 methods to work in preliminarily defined modes 'master' and 'slave'
package appmode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
	"github.com/UnendingLoop/DistributedGrepClone/internal/reader"
	"github.com/docker/distribution/uuid"
)

type taskTotals struct {
	task  *model.MasterTask
	votes int
	data  []string
}

func RunMaster(ctx context.Context, stop context.CancelFunc, ai *model.AppInit) {
	defer stop()
	// прочитать все инпут-строки
	var tasks []*model.MasterTask
	var input []string
	var err error

	// преобразовать вход в задания
	switch len(ai.SearchParam.Source) {
	case 0: // читаем вход из stdIn
		input, err = reader.ReadInput("")
		if err != nil {
			log.Printf("Something went wrong while reading StdIn: %q", err.Error())
			return
		}
		tCTX, cancel := context.WithCancel(ctx)
		tasks = append(tasks, &model.MasterTask{
			TaskID:    uuid.Generate().String(),
			GP:        ai.SearchParam,
			Input:     input,
			CTX:       tCTX,
			CancelCTX: cancel,
		})
	case 1: // читаем из единственного файла
		input, err = reader.ReadInput(ai.SearchParam.Source[0])
		if err != nil {
			log.Printf("Something went wrong while reading file %q: %q", ai.SearchParam.Source[0], err.Error())
			return
		}

		tCTX, cancel := context.WithCancel(ctx)
		tasks = append(tasks, &model.MasterTask{
			TaskID:    uuid.Generate().String(),
			GP:        ai.SearchParam,
			Input:     input,
			CTX:       tCTX,
			CancelCTX: cancel,
		})
	default: // итерируемся по списку файлов
		for _, fname := range ai.SearchParam.Source {
			input, err = reader.ReadInput(fname)
			if err != nil {
				log.Printf("Something went wrong while reading file %q: %q", fname, err.Error())
				return
			}

			tCTX, cancel := context.WithTimeout(ctx, 1*time.Minute)
			tasks = append(tasks, &model.MasterTask{
				TaskID:    uuid.Generate().String(),
				GP:        ai.SearchParam,
				Input:     input,
				FileName:  fname,
				CTX:       tCTX,
				CancelCTX: cancel,
			})
		}
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

func processTasks(ctx context.Context, nodes []string, tasks []*model.MasterTask, quorumN int) ([][]string, error) {
	resCollect := make(chan model.SlaveResult)
	defer close(resCollect)

	// итерируемся по заданиям(их может быть несколько, если на вход подано несколько файлов)
	client := http.Client{}
	for i := range tasks {
		task := tasks[i]
		// итерируемся по всем slave-node адресам и отправляем задания
		for _, nodeAddr := range nodes {
			go sendTaskToNode(task.CTX, &client, nodeAddr, task, resCollect)
		}
	}

	// запускаем сборщика результатов c таймаутом в 1 минуту на сбор
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	return collectResults(ctx, resCollect, tasks, quorumN)
}

func sendTaskToNode(ctx context.Context, client *http.Client, na string, task *model.MasterTask, ch chan<- model.SlaveResult) {
	if !strings.Contains(na, "http://") {
		na = "http://" + na
	}

	// сразу маршалим задание на отправку
	raw, err := json.Marshal(task)
	if err != nil {
		log.Printf("failed to MARSHAL task: %q", err.Error())
		return
	}

	body := bytes.NewReader(raw)

	req, err := http.NewRequestWithContext(ctx, "POST", na+"/task", body)
	if err != nil {
		log.Printf("failed to GENERATE request to slave-node %q: %q", na, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to SEND task to slave-node %q: %q", na, err.Error())
		return
	}

	defer resp.Body.Close()

	var result model.SlaveResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("failed to UNMARSHAL result from slave-node %q: %q", na, err.Error())
		return
	}

	ch <- result
}

func collectResults(ctx context.Context, ch <-chan model.SlaveResult, tasks []*model.MasterTask, quorum int) ([][]string, error) {
	quorumResults := make(map[string]*[]string, len(tasks))

	// готовим мапу задач [TaskID]:*MasterTask чтобы по полученному результату быстро обновлять resMap
	tasksMap := make(map[string]*model.MasterTask)
	for i := range tasks {
		tasksMap[tasks[i].TaskID] = tasks[i]
	}

	// создаем мапу мап для подсчета каждой вариации хеш-суммы по каждому заданию
	resMap := make(map[string]map[uint64]*taskTotals)

	// запуск горутины-сборщика
	wg := sync.WaitGroup{}
	wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case newRes, ok := <-ch:
				if !ok {
					return
				}

				// проверяем, существует ли задача с таким TaskID из полученного результата на стороне мастера
				if _, taskExists := tasksMap[newRes.TaskID]; !taskExists {
					continue
				}

				// создаем результат с полученным TaskID - если его еще нет
				incremented := false
				_, resExists := resMap[newRes.TaskID]
				if !resExists {
					newTT := &taskTotals{
						task:  tasksMap[newRes.TaskID],
						votes: 1,
						data:  newRes.Output,
					}
					incremented = true
					subMap := map[uint64]*taskTotals{newRes.HashSumm: newTT}
					resMap[newRes.TaskID] = subMap
				}

				// создаем запись о полученном HashSumm - если такой еще нет
				submap := resMap[newRes.TaskID]
				_, hashExists := submap[newRes.HashSumm]
				if !hashExists {
					newTT := &taskTotals{
						task:  tasksMap[newRes.TaskID],
						votes: 1,
						data:  newRes.Output,
					}
					incremented = true
					submap[newRes.HashSumm] = newTT
				}

				// проверяем, не достигнут ли уже кворум по полученному HashSumm
				hashRecord := submap[newRes.HashSumm]
				if !incremented {
					hashRecord.votes++
				}
				if hashRecord.votes >= quorum { // если уже достигли кворума - отменяем контекст http-запросов по этой задаче
					hashRecord.task.CancelCTX()
					quorumResults[hashRecord.task.TaskID] = &hashRecord.data
					delete(resMap, newRes.TaskID) // удаляем ключ из мапы результатов, так как уже достигнут кворум
				}
			}
			if len(quorumResults) == len(tasksMap) { // выход из горутины если по всем задачам уже есть кворум-результат
				return
			}
		}
	})

	wg.Wait()

	// проверяем не отменился ли контекст по длине результата
	if len(quorumResults) != len(tasksMap) {
		return nil, errors.New("result collector's context cancelled")
	}

	// формируем результат
	var resStrings [][]string
	for _, v := range tasks {
		lines := quorumResults[v.TaskID]
		if lines != nil {
			resStrings = append(resStrings, *lines)
		}
	}

	// возврат результата
	return resStrings, nil
}
