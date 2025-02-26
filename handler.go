package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
		extraPath := strings.Join(segments[1:], "/")

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
			return
		}

		err = ExecuteScript(scriptName, headers, query, body, w)
		if err != nil {
			switch e := err.(type) {
			case ScriptNotFound:
				w.WriteHeader(404)
				w.Write(fmt.Appendf(nil, "script not found: %s", e.ScriptName))
			case ScriptDisabled:
				w.WriteHeader(403)
				w.Write(fmt.Appendf(nil, "script disabled: %s", e.ScriptName))
			default:
				fmt.Fprintf(os.Stderr, "error executing script %s. error: %s", scriptName, err.Error()) // TODO: log this
				w.WriteHeader(500)
				w.Write([]byte("error executing script"))
			}
		}
	})
	return mux
}
