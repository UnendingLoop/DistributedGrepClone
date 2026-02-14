package transport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
	"github.com/UnendingLoop/DistributedGrepClone/internal/transport"
	"github.com/stretchr/testify/require"
)

type mockProcessor struct {
	returnResultFn func(ctx context.Context, task *model.SlaveTask) *model.SlaveResult
}

func (m mockProcessor) ProcessInput(ctx context.Context, task *model.SlaveTask) *model.SlaveResult {
	return m.returnResultFn(ctx, task)
}

func TestHealthCheck(t *testing.T) {
	srv := transport.NewSlaveServer("", mockProcessor{})
	require.NotEqual(t, nil, srv, "NewSlaveServer returned nil-server")

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	srv.Handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestReceiveTask(t *testing.T) {
	cases := []struct {
		name       string
		mockProcFn *mockProcessor
		ttask      *model.MasterTask
		wantCode   int
	}{
		{
			name: "Positive - successful 200OK",
			mockProcFn: &mockProcessor{
				returnResultFn: func(ctx context.Context, task *model.SlaveTask) *model.SlaveResult {
					return &model.SlaveResult{}
				},
			},
			ttask: &model.MasterTask{
				TaskID: "taskID",
				GP: model.GrepParam{
					Pattern: "pattern",
				},
				Input: []string{},
			},
			wantCode: http.StatusOK,
		},
		{
			name: "Negative - empty task 400BadRequest",
			mockProcFn: &mockProcessor{
				returnResultFn: func(ctx context.Context, task *model.SlaveTask) *model.SlaveResult {
					return &model.SlaveResult{}
				},
			},
			ttask:    nil,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			srv := transport.NewSlaveServer("", tt.mockProcFn)
			require.NotEqual(t, nil, srv, "NewSlaveServer returned nil-server")
			raw, _ := json.Marshal(tt.ttask)
			body := bytes.NewReader(raw)

			req := httptest.NewRequest("POST", "/task", body)
			w := httptest.NewRecorder()

			srv.Handler.ServeHTTP(w, req)

			require.Equal(t, tt.wantCode, w.Code)
		})
	}
}
