package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

func RequestHandler() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Write([]byte("hello there! this is a lazycomm server. thank you and have a nice day!"))
			return
		}

		segments := strings.Split(r.URL.Path, "/")[1:]

		scriptName := segments[0]
		extraPath := "/" + strings.Join(segments[1:], "/")

		headers := make(map[string]string)
		for k, v := range r.Header {
			headers[strings.ToLower(k)] = v[0]
		}

		headers["x-path"] = extraPath
		headers["x-method"] = r.Method

		query := make(map[string]string)
		for k, v := range r.URL.Query() {
			query[strings.ToLower(k)] = v[0]
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("failed to read request body"))
			logrus.Errorf("Failed to read request body: %v", err)
			return
		}

		err = ExecuteScript(scriptName, headers, query, body, w)
		if err != nil {
			switch e := err.(type) {
			case ScriptNotFound:
				w.WriteHeader(404)
				w.Write(fmt.Appendf(nil, "script not found: %s", e.ScriptName))
				logrus.Warnf("tried to access non existent script: %s", e.ScriptName)
			case ScriptDisabled:
				w.WriteHeader(403)
				w.Write(fmt.Appendf(nil, "script disabled: %s", e.ScriptName))
				logrus.Warnf("tried to access disabled script: %s", e.ScriptName)
			case PipeError:
				w.WriteHeader(500)
				w.Write(fmt.Appendf(nil, "failed to create pipe: %s", e.ScriptName))
				logrus.Errorf("failed to create pip %s for script %s", e.PipeName, e.ScriptName)
			case ScriptError:
				w.WriteHeader(500)
				w.Write(fmt.Appendf(nil, "script could not be started: %s", e.ScriptName))
				logrus.Errorf("could not start script %s", e.ScriptName)
			default:
				w.WriteHeader(500)
				w.Write([]byte("error executing script"))
				logrus.Errorf("error executing script %s. error: %s", scriptName, err.Error())
			}
		}
	})
	return mux
}
