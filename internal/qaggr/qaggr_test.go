package qaggr_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
	"github.com/UnendingLoop/DistributedGrepClone/internal/qaggr"
	"github.com/stretchr/testify/require"
)

func TestCollectAggregateResults(t *testing.T) {
	cases := []struct {
		name    string
		testCtx struct {
			ctx    context.Context
			cancel context.CancelFunc
		}
		testCh    chan model.SlaveResult
		testTasks []*model.MasterTask
		testRes   []model.SlaveResult
		testQ     int
		wantErr   string
		wantRes   [][]string
	}{
		{
			name: "Negative - cancelled ctx v1",
			testCtx: func() struct {
				ctx    context.Context
				cancel context.CancelFunc
			} {
				tctx, tcancel := context.WithCancel(context.Background())
				tcancel()
				return struct {
					ctx    context.Context
					cancel context.CancelFunc
				}{
					ctx: tctx, cancel: tcancel,
				}
			}(),
			testCh:    make(chan model.SlaveResult),
			testTasks: []*model.MasterTask{{Task: model.TaskDTO{TaskID: "task1"}}, {Task: model.TaskDTO{TaskID: "task2"}}},
			testQ:     1,
			wantErr:   "exceeded or cancelled without reaching quorum",
			wantRes:   nil,
		},
		{
			name: "Positive - reached quorum",
			testCtx: func() struct {
				ctx    context.Context
				cancel context.CancelFunc
			} {
				tctx, tcancel := context.WithTimeout(context.Background(), 5*time.Second)
				return struct {
					ctx    context.Context
					cancel context.CancelFunc
				}{
					ctx: tctx, cancel: tcancel,
				}
			}(),
			testCh:    make(chan model.SlaveResult),
			testTasks: []*model.MasterTask{{Task: model.TaskDTO{TaskID: "task1"}}, {Task: model.TaskDTO{TaskID: "task2"}}},
			testRes:   []model.SlaveResult{{TaskID: "task1", HashSumm: 300, Output: []string{"1", "2", "3"}}, {TaskID: "task1", HashSumm: 300, Output: []string{"1", "2", "3"}}, {TaskID: "task2", HashSumm: 300, Output: []string{"1", "2", "3"}}, {TaskID: "task2", HashSumm: 300, Output: []string{"1", "2", "3"}}},
			testQ:     2,
			wantErr:   "",
			wantRes:   [][]string{{"1", "2", "3"}, {"1", "2", "3"}},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			go func() {
				for _, v := range tt.testRes {
					tt.testCh <- v
				}
				close(tt.testCh)
			}()

			res, err := qaggr.CollectAggregateResults(tt.testCtx.ctx, tt.testCh, tt.testTasks, tt.testQ)

			tt.testCtx.cancel()

			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr, fmt.Sprintf("received error '%v' instead of '...%v...'", err, tt.wantErr))
			}
			require.Equal(t, tt.wantRes, res, fmt.Sprintf("received result '%v' instead of '%v'", res, tt.wantRes))
		})
	}
}
