package logger

import (
    "log"
	"os"
	"io"
)


func InitLogger(filepath string) (*log.Logger, error){
    file, err := os.Create(filepath + ".log")
    if err != nil {
		return nil, err
    }
    logwriter := io.Writer(file)
	logger := log.New(logwriter, "", log.LstdFlags | log.Lshortfile)
	return logger, nil
}
