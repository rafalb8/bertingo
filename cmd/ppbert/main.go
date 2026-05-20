package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	bert "github.com/rafalb8/bertingo"
)

func main() {
	if len(os.Args) <= 1 {
		fmt.Fprintln(os.Stderr, "Missing argument")
		os.Exit(1)
	}

	if len(os.Args) > 2 {
		length, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Missing failed to parse length")
			os.Exit(1)
		}
		ppbert(os.Args[1], length)
	}

	ppbert(os.Args[1], 0)
}

// pretty print bert
func ppbert(file string, count int) {
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	dec := bert.NewDecoder(f)
	dec.BERT2 = filepath.Ext(file) == ".bert2"

	for b, err := dec.Decode(); err == nil; b, err = dec.Decode() {
		fmt.Println(bert.Tree(b))

		count--
		if count == 0 {
			break
		}
	}
}
