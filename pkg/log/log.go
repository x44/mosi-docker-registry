package log

import (
	"log"
	"os"
)

func Info(prefix, msg string) {
	log.Printf("[INFO] [%s] %s", prefix, msg)
}

func Error(prefix, msg string) {
	log.Printf("[ERROR] [%s] %s", prefix, msg)
}

func Fatal(prefix, msg string) {
	log.Printf("[FATAL] [%s] %s", prefix, msg)
	os.Exit(1)
}
