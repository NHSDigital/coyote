package main

import (
	"os"
	"nhs.uk/coyotecore"
)

func main() {
	coyotecore.Run(os.Args[1:])
}