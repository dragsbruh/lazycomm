package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

// prefix used for initial communication to send/receive metadata between `lazycomm.py` and server
const LZY_PREFIX = "LZY-:"

// ERRORS

type ScriptNotFound struct {
	ScriptName string
}

type ScriptDisabled struct {
	ScriptName string
}

func (e ScriptNotFound) Error() string {
	return "script not found: " + e.ScriptName
}

func (e ScriptDisabled) Error() string {
	return "script disabled: " + e.ScriptName
}

// MAIN STUFF

func ExecuteScript(scriptName string, headers map[string]string, query map[string]string, body []byte, w http.ResponseWriter) error {
	if strings.HasPrefix(scriptName, "_") || strings.HasPrefix(scriptName, ".") {
		return ScriptDisabled{ScriptName: scriptName}
	}
	path := getScriptPath(scriptName)
	if path == "" {
		return ScriptNotFound{ScriptName: scriptName}
	}

	cmd := exec.Command("python", "-u", path)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var capturedStderr strings.Builder

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, LZY_PREFIX) {
				parts := strings.Split(strings.TrimPrefix(line, LZY_PREFIX), " ")
				if len(parts) > 0 {
					command := parts[0]
					switch command {
					case "respond":
						// respond status_code headers_size body_size
						if len(parts) < 4 {
							w.WriteHeader(500)
							w.Write([]byte("invalid respond command sent by script"))
							return
						}
						statusCode, err := strconv.Atoi(parts[1])
						if err != nil {
							w.WriteHeader(500)
							w.Write([]byte("invalid status code sent by script (status code should be an integer)"))
							return
						}
						headersSize, err := strconv.Atoi(parts[2])
						if err != nil {
							w.WriteHeader(500)
							w.Write([]byte("invalid headers size sent by script (header size should be an integer)"))
							return
						}
						bodySize, err := strconv.Atoi(parts[3])
						if err != nil {
							w.WriteHeader(500)
							w.Write([]byte("invalid body size sent by script (body size should be an integer)"))
							return
						}

						headersBuf := make([]byte, headersSize)
						_, err = io.ReadFull(stdout, headersBuf)
						if err != nil {
							w.WriteHeader(500)
							w.Write([]byte("failed to read headers from script"))
							return
						}

						headers := make(map[string]string)
						err = json.Unmarshal(headersBuf, &headers)
						if err != nil {
							w.WriteHeader(500)
							w.Write([]byte("failed to parse json headers from script"))
							return
						}

						bodyBuf := make([]byte, bodySize)
						_, err = io.ReadFull(stdout, bodyBuf)
						if err != nil {
							w.WriteHeader(500)
							w.Write([]byte("failed to read body from script"))
							return
						}
						for k, v := range headers {
							w.Header().Set(k, v)
						}
						w.WriteHeader(statusCode)
						w.Write(bodyBuf)
						return
					}
				}
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			capturedStderr.WriteString(scanner.Text())
			capturedStderr.WriteString("\n")
		}
	}()

	headersJson, err := json.Marshal(headers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal headers: %s", err.Error())
		w.WriteHeader(500)
		w.Write([]byte("failed to marshal headers"))
	}
	queryJson, err := json.Marshal(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal query: %s", err.Error())
		w.WriteHeader(500)
		w.Write([]byte("failed to marshal query"))
	}

	stdin.Write(fmt.Appendf(nil, "%d %d %d\n", len(headersJson), len(queryJson), len(body)))
	stdin.Write(headersJson)
	stdin.Write(queryJson)
	stdin.Write(body)

	stdin.Close()

	if err := cmd.Wait(); err != nil {
		writeLog(scriptName, capturedStderr.String())
		w.WriteHeader(500)
		w.Write(fmt.Appendf(nil, "script exited with exit code %d. stderr was logged", cmd.ProcessState.ExitCode()))
		return nil
	}

	return nil
}

// UTILITIES

func getScriptPath(scriptName string) string {
	possibleNames := []string{
		scriptName + ".py",
		"." + scriptName + ".py",
	}
	for _, name := range possibleNames {
		fullPath := path.Join(".", "scripts", name)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}
	return ""
}

func writeLog(scriptName string, log string) {
	os.MkdirAll(LogsDir, 0755)
	logPath := path.Join(LogsDir, scriptName+".log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file for writing: %s", err.Error())
		return
	}
	defer f.Close()
	_, err = f.WriteString(log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write to log file: %s", err.Error())
		return
	}
	f.WriteString("\n-----------------------------\n")
}
