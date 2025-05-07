package main

import (
	"github.com/synexism/synexis/cmd/synexis"
)

func init() {
	synexis.Initialize()
}

func main() {
	synexis.Execute()
}
