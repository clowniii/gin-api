package logs

import (
	"log"
	"os"
)

var logger *log.Logger

func InitLogger() {
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	logger = log.New(file, "", log.LstdFlags|log.Lshortfile)
}

func Info(v ...interface{}) {
	logger.Println("[INFO]", v)
}

func Error(v ...interface{}) {
	logger.Println("[ERROR]", v)
}
