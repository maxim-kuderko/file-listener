package main

import (
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"strconv"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(&lumberjack.Logger{
		Filename:   "/home/max/Projects/src/github.com/maxim-kuderko/file-listener/example/log-stream.log",
		MaxSize:    10, // megabytes
		MaxBackups: 300,
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	})
	for i := 0; i < 100000000; i++ {
		log.Println("{\"key\": \"" + strconv.Itoa(i) + "\"}")
	}
}
