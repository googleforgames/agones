package main

import (
	"log"
)

func main() {
	log.Println("Checking that modified examples have new versions.")
	log.Println()

	failed := getFailedFiles()

	logReport(failed)
}
