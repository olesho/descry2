// log
package descry

import (
	"fmt"
	"log"
	"os"
)

const (
	LEVEL_NONE       = 0
	LEVEL_DEBUG      = 1
	LEVEL_PRODUCTION = 2
)

type Logger struct {
	Level      int
	fileLogger *log.Logger
}

func NewLogger() *Logger {
	file, _ := os.OpenFile("error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	return &Logger{
		Level:      LEVEL_DEBUG,
		fileLogger: log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

// Handle error and log it
func (l *Logger) IsError(message string, err error) bool {
	if err != nil {
		switch l.Level {
		case LEVEL_DEBUG:
			fmt.Println(message)
			fmt.Println(err)
			l.fileLogger.Println(err)
			break
		case LEVEL_PRODUCTION:
			l.fileLogger.Println(err)
		}
		return true
	}
	return false
}

func (l *Logger) Message(msg ...interface{}) {
	switch l.Level {
	case LEVEL_DEBUG:
		fmt.Println(msg)
		l.fileLogger.Println(msg)
		break
	case LEVEL_PRODUCTION:
		l.fileLogger.Println(msg)
	}
}
