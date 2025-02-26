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

	"github.com/sirupsen/logrus"
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

type PipeError struct {
	ScriptName string
	PipeName   string
}

type ScriptError struct {
	ScriptName string
}

func (e ScriptNotFound) Error() string {
	return "script not found: " + e.ScriptName
}

func (e ScriptDisabled) Error() string {
	return "script disabled: " + e.ScriptName
}

func (e PipeError) Error() string {
	return "failed to create pipe " + e.PipeName + ": " + e.ScriptName
}

func (e ScriptError) Error() string {
	return "failed to execute script: " + e.ScriptName
}

// MAIN STUFF

// should not return error when response is already sent. return nil instead
func ExecuteScript(scriptName string, headers map[string]string, query map[string]string, body []byte, w http.ResponseWriter) error {
	if strings.HasPrefix(scriptName, "_") || strings.HasPrefix(scriptName, ".") {
		return ScriptDisabled{ScriptName: scriptName}
	}
	scriptPath := getScriptPath(scriptName)
	if scriptPath == "" {
		disabledPath := path.Join(".", "scripts", "_"+scriptName+".py")
		if _, err := os.Stat(disabledPath); err == nil {
			return ScriptDisabled{ScriptName: scriptName}
		}
		return ScriptNotFound{ScriptName: scriptName}
	}

	cmd := exec.Command("python", "-u", scriptPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return PipeError{ScriptName: scriptName, PipeName: "stdin"}
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return PipeError{ScriptName: scriptName, PipeName: "stdout"}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return PipeError{ScriptName: scriptName, PipeName: "stderr"}
	}

	if err := cmd.Start(); err != nil {
		return ScriptError{ScriptName: scriptName}
	}

	logrus.Infof("started script %s", scriptName)

	var capturedStderr strings.Builder

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, LZY_PREFIX) {
				statusCode := handleScriptResponse(strings.TrimPrefix(line, LZY_PREFIX), stdout, w)
				if statusCode == -10 {
					logrus.Errorf("response parse error for %s", scriptName)
				} else {
					logrus.Infof("script %s responded with status code %d", scriptName, statusCode)
				}
				return
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
		logrus.Errorf("failed to marshal headers: %s", err.Error())
		w.WriteHeader(500)
		w.Write([]byte("failed to marshal headers"))
		return nil
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		logrus.Errorf("failed to marshal query: %s", err.Error())
		w.WriteHeader(500)
		w.Write([]byte("failed to marshal query"))
		return nil
	}

	stdin.Write(fmt.Appendf(nil, "%d %d %d\n", len(headersJson), len(queryJson), len(body)))
	stdin.Write(headersJson)
	stdin.Write(queryJson)
	stdin.Write(body)

	stdin.Close()

	if err := cmd.Wait(); err != nil {
		logStderr(scriptName, capturedStderr.String())
		logrus.Errorf("script exited with exit code %d. stderr was logged", cmd.ProcessState.ExitCode())

		w.WriteHeader(500)
		w.Write(fmt.Appendf(nil, "script exited with exit code %d. stderr was logged", cmd.ProcessState.ExitCode()))
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

func logStderr(scriptName string, log string) {
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

func handleScriptResponse(metadata string, stdout io.Reader, w http.ResponseWriter) int {
	parts := strings.Split(metadata, " ")

	if len(parts) < 3 {
		w.WriteHeader(500)
		w.Write([]byte("invalid respond command sent by script"))
		return -10
	}

	statusCodeText := parts[0]
	headersSizeText := parts[1]
	bodySizeText := parts[2]

	statusCode, ok := parseIntOrFail(statusCodeText, "status code", w)
	if !ok {
		return -10
	}
	headersSize, ok := parseIntOrFail(headersSizeText, "headers size", w)
	if !ok {
		return -10
	}
	bodySize, ok := parseIntOrFail(bodySizeText, "body size", w)
	if !ok {
		return -10
	}

	headersBuf := make([]byte, headersSize)
	_, err := io.ReadFull(stdout, headersBuf)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("failed to read headers from script"))
		return -10
	}

	bodyBuf := make([]byte, bodySize)
	_, err = io.ReadFull(stdout, bodyBuf)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("failed to read body from script"))
		return -10
	}

	headers := make(map[string]string)
	err = json.Unmarshal(headersBuf, &headers)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("failed to parse json headers from script"))
		return -10
	}

	for k, v := range headers {
		w.Header().Set(k, v)
	}

	w.WriteHeader(statusCode)
	w.Write(bodyBuf)

	return statusCode
}

func parseIntOrFail(value string, title string, w http.ResponseWriter) (int, bool) {
	num, err := strconv.Atoi(value)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf(nil, "invalid %s sent by script (%s should be an integer)", title, title))
		return 0, false
	}
	return num, true
}
