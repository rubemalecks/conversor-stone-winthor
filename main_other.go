//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	for _, a := range os.Args[1:] {
		r, err := Converter(a)
		if err == nil {
			err = r.GravarTodos(filepath.Dir(a))
		}
		if err != nil {
			fmt.Println("ERRO:", err)
			continue
		}
		fmt.Println(r.Resumo())
		fmt.Println(r.AvisoDatas())
		fmt.Println(r.Motivos)
	}
}
