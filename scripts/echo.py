import json
import sys

# standalone example that does not depend on lazycomm.py
# this should be used for testing purposes only

lzy_prefix = "LZY-:"

# read metadata
line = sys.stdin.buffer.readline().decode()
headers_size, query_size, body_size = map(int, line.split())

# read headers query body based on given metadata
headers = json.loads(sys.stdin.buffer.read(headers_size).decode())
query = json.loads(sys.stdin.buffer.read(query_size).decode())
body = sys.stdin.buffer.read(body_size)

# merge query and headers for response
for k, v in query.items():
    headers[k] = v
    
# prepare response
resheaders = json.dumps(headers).encode()
resbody = body

status_code = 200

# send response
sys.stdout.buffer.write(f"{lzy_prefix}{status_code} {len(resheaders)} {len(resbody)}\n".encode())
sys.stdout.buffer.write(resheaders)
sys.stdout.buffer.write(resbody)
sys.stdout.buffer.flush()
