# helper script that parses and sends data to lazycomm server
# include this module in your script before anything else

import json
import sys

_lzy_prefix = "LZY-:"

_meta_line = sys.stdin.buffer.readline().decode().strip()
_header_size, _query_size, _body_size = map(int, _meta_line.split())

headers = json.loads(sys.stdin.buffer.read(_header_size).decode())
query = json.loads(sys.stdin.buffer.read(_query_size).decode())
body = sys.stdin.buffer.read(_body_size)

path = headers["x-path"]
method = headers["x-method"]

def respond(status_code: int, headers: dict[str, str], body: bytes):
    res_headers = json.dumps(headers).encode()
    sys.stdout.buffer.write(f"{_lzy_prefix}{status_code} {len(res_headers)} {len(body)}\n".encode())
    sys.stdout.buffer.write(res_headers)
    sys.stdout.buffer.write(body)
    sys.stdout.flush()
    