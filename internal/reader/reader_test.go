package reader_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/UnendingLoop/DistributedGrepClone/internal/reader"
	"github.com/stretchr/testify/require"
)

func TestReadInput(t *testing.T) {
	inputLine := "line1\nline2\nline3\nline4"
	wantOutput := []string{"line1", "line2", "line3", "line4"}

	cases := []struct {
		name    string
		stdin   io.Reader
		isDir   bool
		isReal  bool // false только для кейса "файл не найден"
		input   string
		wantErr string
	}{
		{
			name:    "Positive - 0 files stdIn",
			stdin:   bytes.NewReader([]byte(inputLine)),
			isDir:   false,
			isReal:  true,
			input:   inputLine,
			wantErr: "",
		},
		{
			name:    "Positive - 1 file",
			stdin:   nil,
			isDir:   false,
			isReal:  true,
			input:   inputLine,
			wantErr: "",
		},
		{
			name:    "Negative - file is a directory",
			stdin:   nil,
			isDir:   true,
			isReal:  true,
			input:   "",
			wantErr: "is a directory",
		},
		{
			name:    "Negative - file not found",
			stdin:   nil,
			isDir:   false,
			isReal:  false,
			input:   "",
			wantErr: "system cannot find the file specified",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			fileName := ""
			switch {
			case !tt.isReal:
				fileName = "test_unreal_file_12345.txt"
			case tt.stdin == nil:
				fileName = createTempFile(t, tt.input, tt.isDir)
			}

			res, err := reader.ReadInput(tt.stdin, fileName)

			switch tt.wantErr {
			case "":
				require.Equal(t, wantOutput, res, "Output array is not equal to wantOutput")
			default:
				require.ErrorContains(t, err, tt.wantErr, fmt.Sprintf("Received error %v doesn't contain %q", err, tt.wantErr))
			}
		})
	}
}

// вспомогательная функция для создания временного файла
func createTempFile(t *testing.T, content string, isDir bool) string {
	t.Helper()
	switch isDir {
	case true:
		return t.TempDir()
	default:
		f, err := os.CreateTemp("", "mygrep_test_*.txt")
		if err != nil {
			t.Fatalf("failed to create temp-file: %v", err)
		}
		if _, err := f.WriteString(content); err != nil {
			t.Fatalf("failed to write provided content to temp-file: %v", err)
		}
		f.Close()
		return f.Name()
	}
}
