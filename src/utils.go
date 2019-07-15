package main

import (
	"os"
	"strconv"
)

func GetHostindex() int {
	stringa, _ := os.Hostname()
	stringa = stringa[15:17]

	index, _ := strconv.Atoi(stringa)
	return index - 1
}
