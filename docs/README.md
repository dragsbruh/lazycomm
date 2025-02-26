# lazycomm

lazycomm is a lightweight HTTP server for running scripts as "webhooks." it was initially created to enable communication between browser userscripts and python scripts.

> for detailed info on how communication is handled, see [comms.md](./comms.md)

## usage

1. place your python scripts in the `scripts` directory. to prevent pushing them to the repo, prefix script names with `.`.
2. prefixing script names with `_` will prevent them from being run, and also prevent them from being pushed to the repo.
3. install your userscripts in your browser and ensure they use the correct port (default is `6565`).
4. run `go run main.go` to start the server. you can also run the compiled binary, but ensure the `scripts` directory is in the same location as the binary.
5. the server will listen for webhooks at `http://localhost:6565/{script_name}`.

## notes

- `script_name` is the file name without the `.py` extension. for example:
  - if your script is `foo.py`, the webhook will be `foo`.
  - if both `foo.py` and `.foo.py` exist, `foo.py` is given priority and `.foo.py` cannot be invoked in any way.
  - `_foo.py` cannot be invoked at all, scripts starting with `_` are considered "disabled" and cannot be run.
  - similarly, `._foo.py` cannot be invoked at all. since any of the `.` and `_` prefixes prevent pushing to the repo, it is not recommended to name your scripts this way.

- **do not** run the server as administrator. since python scripts execute with the same privileges as the server, an attacker with initial access could run arbitrary code with elevated permissions. the server allows this by default but will warn you if it detects administrator privileges.

## writing scripts

scripts are written in python and must be named with the `.py` extension.

```python
from lazycomm import path, query, headers, body, respond

# lazycomm.py is a helper module provided in the scripts directory by default.

# absolutely do not read anything from stdin before importing `lazycomm`.
# this is because the server sends request data to the script via stdin.
# lazycomm.py parses this data for you.

# path -> full path sent to the webhook (string)
# query -> query params sent to the webhook (dictionary with lowercase keys)
# headers -> headers sent to the webhook (dictionary with lowercase keys)
# body -> raw body sent to the webhook (bytes)
# respond -> function to send a response.
#           takes the following arguments:
#           - status_code (int)
#           - headers (dictionary with lowercase keys)
#           - body (bytes)
#     only call `respond` once. any other calls will be ignored.

respond(200, {"content-type": "text/plain"}, b"hello there!")
# if you do not call respond at all, the server will respond with a 200 ok after script exits.
# if respond is not called and script exits with non zero code, the server responds with a 500 internal server error.
# any code after respond will still be executed, except now the connection to initial request is closed

print("hello there!") # this will still run but will not do anything. stdout is not captured.

import sys # you can import any modules that exist in the python environment.

sys.stderr.write("this will be captured as an error.") # will only be logged if the script exits with a non-zero exit code.
exit(1) # will be logged as an error.
```
