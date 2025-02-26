import json
import sys

# standalone example that does not depend on lazycomm.py
# this should be used for testing purposes only

lzy_prefix = "LZY-:"

# read metadata
line = sys.stdin.readline()

# parse metadata to headers size query size and body size
headers_size = int(line.split(" ")[0])
query_size = int(line.split(" ")[1])
body_size = int(line.split(" ")[2])

# read headers query body based on given metadata
headers = json.loads(sys.stdin.read(headers_size))
query = json.loads(sys.stdin.read(query_size))
body = sys.stdin.read(body_size)

# merge query and headers for response
for k, v in query.items():
    headers[k] = v
    
# prepare response
resheaders = json.dumps(headers).encode()
resbody = body.encode()

status_code = 200

# send response
sys.stdout.buffer.write(f"{lzy_prefix}respond {status_code} {len(resheaders)} {len(resbody)}\n".encode())
sys.stdout.buffer.write(resheaders)
sys.stdout.buffer.write(resbody)
sys.stdout.buffer.flush()
