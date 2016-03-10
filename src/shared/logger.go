package logger

import (
    "log"
)

type logS struct {
    log *log.Logger
}

func (logS) Init(filepath string) {
    file, err := os.Create(filepath)
    if err != nil {
        log.Fatal(err)
    }
    logw := io.Writer(file)
    logS.log = log.New(logw, "*", log.Lshortfile)
}
