// Package output is responsible for printing result to stdIn according to SearchParam values
package output

import (
	"fmt"

	"grepClone/internal/parser"
)

// учесть что нужно делать префикс имени файла если файлов >1  + нумерация строк

func PrintLine(line, fileName string, n int) {
	switch {
	case fileName == "":
		fmt.Println(line)
	case parser.SP.PrintFileName && parser.SP.EnumLine:
		fmt.Printf("%s: %d: %s\n", fileName, n, line)
	case parser.SP.EnumLine:
		fmt.Printf("%d: %s\n", n, line)
	case parser.SP.PrintFileName:
		fmt.Printf("%s: %s\n", fileName, line)
	default:
		fmt.Println(line)
	}
}
