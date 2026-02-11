package processor_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"grepClone/internal/model"
	"grepClone/internal/parser"
	"grepClone/internal/processor"
)

// вспомогательная функция для перехвата stdout
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return strings.TrimSpace(buf.String())
}

// вспомогательная функция для создания временного файла
func createTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "grepClone_test_*.txt")
	if err != nil {
		t.Fatalf("не удалось создать временный файл: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("не удалось записать во временный файл: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestProcessInput_CountOnly(t *testing.T) {
	file := createTempFile(t, "foo\nbar\nfoo\nbaz\nfoo")
	defer os.Remove(file)

	parser.SP = model.SearchParam{} // сброс глобального состояния
	parser.SP.Pattern = "foo"
	parser.SP.ExactMatch = true
	parser.SP.CountFound = true
	parser.SP.PrintFileName = false

	parser.SP.RegexpPattern = nil // на всякий случай

	got := captureOutput(t, func() {
		err := processor.ProcessInput(file)
		if err != nil {
			t.Fatalf("ProcessInput вернул ошибку: %v", err)
		}
	})

	want := "3" // foo трижды
	if got != want {
		t.Errorf("ожидался вывод %q, а получили %q", want, got)
	}
}

func TestProcessInput_BeforeAfterContext(t *testing.T) {
	file := createTempFile(t, "1\n2\nfoo\n4\n5\n6\nfoo\n8\n9")
	defer os.Remove(file)

	parser.SP.Pattern = "foo"
	parser.SP.ExactMatch = true
	parser.SP.CountFound = false
	parser.SP.CtxBefore = 1
	parser.SP.CtxAfter = 1
	parser.SP.EnumLine = false
	parser.SP.PrintFileName = false

	got := captureOutput(t, func() {
		err := processor.ProcessInput(file)
		if err != nil {
			t.Fatalf("ProcessInput вернул ошибку: %v", err)
		}
	})

	// ожидаем контекст: строки до и после совпадения
	// для первой foo → 2, foo, 4
	// для второй foo → 6, foo, 8
	want := strings.Join([]string{"2", "foo", "4", "--", "6", "foo", "8"}, "\n")

	if got != want {
		t.Errorf("ожидался вывод:\n%s\nа получили:\n%s", want, got)
	}
}

func TestProcessInput_FileDoesNotExist(t *testing.T) {
	parser.SP.CountFound = false
	err := processor.ProcessInput("no_such_file_12345.txt")
	if err == nil {
		t.Error("ожидалась ошибка при несуществующем файле, но её не было")
	}
}

func TestProcessInput_SourceIsDirectory(t *testing.T) {
	dir := t.TempDir()
	parser.SP.CountFound = false
	err := processor.ProcessInput(dir)
	if err == nil {
		t.Error("ожидалась ошибка при указании директории как источника, но её не было")
	}
}
