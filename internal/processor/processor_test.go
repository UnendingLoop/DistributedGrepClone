package processor_test

import (
	"context"
	"testing"

	"github.com/UnendingLoop/DistributedGrepClone/internal/model"
	"github.com/UnendingLoop/DistributedGrepClone/internal/processor"
	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/require"
)

func TestProcessInput(t *testing.T) {
	inputArray := []string{"abcabcabc123", "abcabc123", "abc123", "123"}
	cases := []struct {
		name    string
		task    *model.SlaveTask
		wantRes *model.SlaveResult
		ctx     context.Context
	}{
		{
			name: "Positive - count matching lines regexp",
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:    "^abc",
					ExactMatch: false,
					CountFound: true,
				},
				Input: inputArray,
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{"3\n"},
				HashSumm: hasher(t, []string{"3\n"}),
			},
			ctx: context.Background(),
		},
		{
			name: "Positive - count matching lines non-regexp",
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:    "abc",
					ExactMatch: true,
					CountFound: true,
				},
				Input: inputArray,
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{"3\n"},
				HashSumm: hasher(t, []string{"3\n"}),
			},
			ctx: context.Background(),
		},
		{
			name: "Negative - cancelled context - std flow",
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:    "abc",
					ExactMatch: true,
					CountFound: false,
				},
				Input: inputArray,
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{},
				HashSumm: hasher(t, []string{}),
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
		},
		{
			name: "Negative - cancelled context - countLines",
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:    "abc",
					ExactMatch: true,
					CountFound: true,
				},
				Input: inputArray,
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{},
				HashSumm: hasher(t, []string{}),
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
		},
		{
			name: "Positive - ignore case",
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:    "(?i)^ABC",
					IgnoreCase: true,
				},
				Input: inputArray,
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{"abcabcabc123", "abcabc123", "abc123"},
				HashSumm: hasher(t, []string{"abcabcabc123", "abcabc123", "abc123"}),
			},
			ctx: context.Background(),
		},
		{
			name: "Positive - invert result",
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:      "ABC$",
					InvertResult: true,
				},
				Input: inputArray,
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{"abcabcabc123", "abcabc123", "abc123", "123"},
				HashSumm: hasher(t, []string{"abcabcabc123", "abcabc123", "abc123", "123"}),
			},
			ctx: context.Background(),
		},
		{
			name: "Positive - enum lines",
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:  "abc",
					EnumLine: true,
				},
				Input: inputArray,
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{"1:abcabcabc123", "2:abcabc123", "3:abc123"},
				HashSumm: hasher(t, []string{"1:abcabcabc123", "2:abcabc123", "3:abc123"}),
			},
			ctx: context.Background(),
		},
		{
			name: "Positive - print filename",
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:       "abc",
					PrintFileName: true,
				},
				Input:    inputArray,
				FileName: "someName",
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{"someName:abcabcabc123", "someName:abcabc123", "someName:abc123"},
				HashSumm: hasher(t, []string{"someName:abcabcabc123", "someName:abcabc123", "someName:abc123"}),
			},
			ctx: context.Background(),
		},
		{
			name: "Positive - enum lines & print filename",
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:       "abc",
					EnumLine:      true,
					PrintFileName: true,
				},
				Input:    inputArray,
				FileName: "someName",
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{"someName:1:abcabcabc123", "someName:2:abcabc123", "someName:3:abc123"},
				HashSumm: hasher(t, []string{"someName:1:abcabcabc123", "someName:2:abcabc123", "someName:3:abc123"}),
			},
			ctx: context.Background(),
		},
		{
			name: "Positive - print ctx after",
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:  "^123$",
					CtxAfter: 5,
				},
				Input: inputArray,
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{"123"},
				HashSumm: hasher(t, []string{"123"}),
			},
			ctx: context.Background(),
		},
		{
			name: "Positive - print ctx before",
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:   "^123$",
					CtxBefore: 2,
				},
				Input: inputArray,
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{"abcabc123", "abc123", "123"},
				HashSumm: hasher(t, []string{"abcabc123", "abc123", "123"}),
			},
			ctx: context.Background(),
		},
		{
			name: "Positive - print ctx circle", // круговой контекст реализован через A и B на этапе парсинга аргументов
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:   "^abc1",
					CtxAfter:  1,
					CtxBefore: 1,
				},
				Input: inputArray,
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{"abcabc123", "abc123", "123"},
				HashSumm: hasher(t, []string{"abcabc123", "abc123", "123"}),
			},
			ctx: context.Background(),
		},
		{
			name: "Positive - count lines & print filename",
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:       "abc",
					CountFound:    true,
					PrintFileName: true,
				},
				Input:    inputArray,
				FileName: "someName",
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{"someName:3"},
				HashSumm: hasher(t, []string{"someName:3"}),
			},
			ctx: context.Background(),
		},
		{
			name: "Negative - broken regexp",
			task: &model.SlaveTask{
				TaskID: "testTask",
				GP: model.GrepParam{
					Pattern:       "?abc",
					CountFound:    true,
					PrintFileName: true,
				},
				Input:    inputArray,
				FileName: "someName",
			},
			wantRes: &model.SlaveResult{
				TaskID:   "testTask",
				Output:   []string{},
				HashSumm: hasher(t, []string{}),
			},
			ctx: context.Background(),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			test := processor.Processor{}

			res := test.ProcessInput(tt.ctx, tt.task)

			require.Equal(t, tt.wantRes, res)
		})
	}
}

func hasher(t *testing.T, input []string) uint64 {
	t.Helper()
	hs := xxhash.New()
	for _, s := range input {
		_, err := hs.WriteString(s)
		require.NoError(t, err, "failed to write data to count hash")
	}

	return hs.Sum64()
}
