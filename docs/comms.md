# **communication between lazycomm and script**

lazycomm and the script communicate through **stdin/stdout** in two separate phases:

## **server to script**

when the server starts the script, it sends:

1. **metadata (space-separated lengths in bytes)**:
   ```
   <headers json length> <query params json length> <body length>
   ```
2. **data (sent in order):**
   - json-encoded **headers**
   - json-encoded **query parameters**
   - raw **body**

after sending this, the server waits for a response from the script.

---

## **2. script to server**

the script should respond with:

1. **metadata (space-separated values):**
   ```
   <status code> <headers json length> <body length>
   ```
2. **data (sent in order):**
   - json-encoded **headers**
   - raw **body**

after receiving this, the server:

- **continues capturing stderr** for errors.
- **logs** any non-zero exit codes from the script.

in the event that the script exits without sending a response, the server will respond to the client with a generic `200 ok` response.

### extras

- all request headers and query params are converted to lowercase before being sent to the script
- the full request path is also sent as a header ("x-path")
- the method is also sent as a header ("x-method")

### tips

if your script is supposed to not respond, you can send an early response. this will not leave the connection open, but will allow your script to continue running.
