package main

import (
	"io"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/pelletier/go-toml/v2"
	"github.com/sirupsen/logrus"
)

const LogsDir string = ".log"

var Port int

func main() {
	LoadConfig()
	os.MkdirAll(LogsDir, 0755)

	logFile, err := os.OpenFile(path.Join(LogsDir, "server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logrus.Fatalf("Failed to open log file: %v", err)
	}

	logrus.SetOutput(io.MultiWriter(os.Stdout, logFile))

	server := http.Server{
		Addr:    "127.0.0.1:" + strconv.Itoa(Port),
		Handler: RequestHandler(),
	}

	logrus.Infof("Server started on port %d", Port)
	server.ListenAndServe()
}

func LoadConfig() {
	var err error
	var conf struct {
		Port int `toml:"port"`
	}

	confData, err := os.ReadFile("config.toml")
	if err != nil {
		Port = 6565
		return
	}
	err = toml.Unmarshal(confData, &conf)
	if err != nil {
		panic(err)
	}

	Port = conf.Port
}
