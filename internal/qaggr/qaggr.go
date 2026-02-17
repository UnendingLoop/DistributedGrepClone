// Package qaggr - provides method to aggregate all results received from slave-nodes and reach quorum
package qaggr

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
)

type taskTotals struct {
	task  *model.MasterTask
	votes int
	data  []string
}

func CollectAggregateResults(ctx context.Context, ch <-chan model.SlaveResult, tasks []*model.MasterTask, quorum int) ([][]string, error) {
	quorumResults := make(map[string]*[]string, len(tasks))

	// готовим мапу задач [TaskID]:*MasterTask чтобы по полученному результату быстро обновлять resMap
	tasksMap := make(map[string]*model.MasterTask)
	for i := range tasks {
		tasksMap[tasks[i].Task.TaskID] = tasks[i]
	}

	// создаем мапу мап для подсчета каждой вариации хеш-суммы по каждому заданию
	resMap := make(map[string]map[uint64]*taskTotals)

	// закладываем очистку используемых мап при выходе
	defer func() {
		for k := range resMap {
			delete(resMap, k)
		}
		for k := range tasksMap {
			delete(tasksMap, k)
		}
		for k := range quorumResults {
			delete(quorumResults, k)
		}
	}()

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
				if hashRecord.votes >= quorum { // если уже достигли кворума - отменяем контекст по этой задаче
					if hashRecord.task.CancelCTX != nil {
						hashRecord.task.CancelCTX()
					}
					quorumResults[hashRecord.task.Task.TaskID] = &hashRecord.data
					delete(resMap, newRes.TaskID) // удаляем ключ из мапы результатов, так как уже достигнут кворум
				}
			default:
				if len(quorumResults) == len(tasksMap) { // выход из горутины, если по завершении принятия результатов канал не закрылся
					return
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
	})

	wg.Wait()

	// формируем результат - в него попадут только задачи, достигшие кворума по результатам
	var resStrings [][]string
	for _, v := range tasks {
		select {
		case <-ctx.Done():
			return nil, errors.New("CollectAggregateResults's context cancelled on the stage of forming a final result")
		default:
			lines, ok := quorumResults[v.Task.TaskID]
			if !ok {
				log.Printf("Quorum failed for file %q", v.Task.FileName)
				continue
			}
			if lines != nil {
				resStrings = append(resStrings, *lines)
			}
		}
	}

	// возврат результата
	return resStrings, nil
}
