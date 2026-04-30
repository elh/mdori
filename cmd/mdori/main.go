package main

import (
	"os"

	"github.com/elh/mdori/internal/mdori"
)

func main() {
	os.Exit(mdori.Main(os.Args[1:], os.Stdout, os.Stderr))
}
