package main

import (
	"os"
	"strconv"
	"time"
)

func main() {
	f, _ := os.OpenFile("/Users/max/projects/src/github.com/maxim-kuderko/file-listener/example/log-stream.log", os.O_APPEND|os.O_WRONLY, 0600)
	for i := 0; i < 100000000; i++ {
		f.WriteString("{\"key\": \"" + strconv.Itoa(i) + "\"}\n")
		time.Sleep(time.Millisecond * 10)
	}
}
