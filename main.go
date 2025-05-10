package main

import (
	"github.com/synxms/synexis/cmd/synexis"
)

func init() {
	synexis.Initialize()
}

func main() {
	synexis.Execute()
}
