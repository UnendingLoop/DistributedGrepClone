package output

import (
	"io"
	"os"
	"strings"
	"testing"

	"grepClone/internal/parser"
)

func Test_PrintLine(t *testing.T) {
	cases := []struct {
		name          string
		text          string
		expResult     string
		printFileName bool
		printLineN    bool
		lineN         int
		fileName      string
	}{
		{
			name:          "Separating block print",
			text:          "--",
			expResult:     "--",
			printFileName: false,
			fileName:      "",
			printLineN:    false,
			lineN:         0,
		},
		{
			name:          "filename + line POS case",
			text:          "test_line",
			expResult:     "testName: test_line",
			printFileName: true,
			fileName:      "testName",
			printLineN:    false,
		},
		{
			name:          "filename + line NEG case",
			text:          "test_line",
			expResult:     "test_line",
			printFileName: false,
			fileName:      "testName",
			printLineN:    false,
		},
		{
			name:          "filename + line NEG case",
			text:          "test_line",
			expResult:     "test_line",
			printFileName: true,
			fileName:      "",
			printLineN:    false,
		},
		{
			name:          "filename + lineN + line POS case",
			text:          "testLine",
			expResult:     "testname: 777: testLine",
			printFileName: true,
			fileName:      "testname",
			printLineN:    true,
			lineN:         777,
		},
		{
			name:          "filename + lineN + line NEG case",
			text:          "testLine",
			expResult:     "testLine",
			printFileName: false,
			fileName:      "testname",
			printLineN:    false,
			lineN:         777,
		},
		{
			name:          "lineN + line pos case",
			text:          "testLine",
			expResult:     "777: testLine",
			printFileName: false,
			fileName:      "testName",
			printLineN:    true,
			lineN:         777,
		},
		{
			name:          "lineN + line neg case",
			text:          "testLine",
			expResult:     "testLine",
			printFileName: false,
			printLineN:    false,
			lineN:         777,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// делаем перехват вывода
			oldStdOut := os.Stdout
			defer func() { os.Stdout = oldStdOut }()
			tReader, tWriter, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create output redirection pipe: %v", err)
			}
			os.Stdout = tWriter

			// подставляем зависимость
			parser.SP.PrintFileName = tc.printFileName
			parser.SP.EnumLine = tc.printLineN

			PrintLine(tc.text, tc.fileName, tc.lineN)
			tWriter.Close()

			outputBytes, err := io.ReadAll(tReader)
			if err != nil {
				t.Fatalf("Failed to read from pipe: %v", err)
			}
			actual := strings.TrimSpace(string(outputBytes))

			if actual != tc.expResult {
				t.Errorf("%q failed: \nNeed to print filename: %v\nSpecified filename: %q\nNeed to print line number: %v\nSpecified line number: %d\nInput line: %q\nExpected: %q\nActual: %q\n", tc.name, tc.printFileName, tc.fileName, tc.printLineN, tc.lineN, tc.text, tc.expResult, actual)
			}
		})
	}
}
