package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"encoding/json"
)

var websocketHTML = `
<html>
    <head>
        <title>websocket</title>
    </head>
    <style>
    #console {
         font-family: monospace;
         font-weight: bold;
         line-height: 1.5em;
         border-top: 1px dashed lightgray;
    }

    #console div {
        border-bottom: 1px dashed lightgray;
    }

    #console div:before {
        display: inline-block;
        width: 5em;
    }

    #console div.send, div.recv {
        font-weight: normal;
        color: gray;
    }

    #console div.info:before {
        color: black;
        content: "[info]";
    }

    #console div.error:before {
        color: red;
        content: "[error]";
    }

    #console div.send:before {
        color: blue;
        content: "[send]";
    }

    #console div.recv:before {
        color: green;
        content: "[recv]";
    }

    .hidden {
        display: none;
    }

    button {
        border-radius: 0.3em;
        border: 1px solid lightgray;
        background: white;
    }

    #msg {
        margin-top: 0.5em;
        text-align: right;
    }

    #msg textarea {
        text-align: left;
        border-radius: 0.3em;
        border: 1px solid lightgray;
        width: 100%;
        display: block;
        min-height: 8em;
        min-width: 20em;
    }

    #msg button {
        margin-top: 0.5em;
    }

    #panel {
        position: fixed;
        top: 1em;
        right: 1em;

        border: 1px solid lightgray;
        border-radius: 0.3em;
        background: white;
        padding: 0.5em;
    }

    </style>
    <body>
        <div id="panel" />
            <div>
                <button id="pause" class="hidden">Pause Messaging</button>
                <button id="resume" class="hidden">Resume Messaging</button>
                <button id="connect" class="hidden">Connect to Server</button>
                <button id="disconnect" class="hidden">Disconnect from Server</button>
                <button id="cancel" class="hidden">Cancel Connection Attempt</button>
            </div>
            <div id="msg" class="hidden">
                <textarea id="content"></textarea>
                <button id="send">Send Message</button>
            </div>
        </div>
        <div id="console" />
        <script>
            var ws
            var messageDelay = 1500
            var connectDelay = 5000
            var autoReconnect = true;

            function log(text, classes) {
                var node = document.createElement("div");
                node.textContent = text;
                node.className = classes
                document.getElementById('console').appendChild(node);
                window.scrollTo(0,document.body.scrollHeight);
            }

            var messageTimer = null
            var connectTimer = null
            var counter = 0

            function send() {
                var data = counter + ' = 0x' + counter.toString(16);
                ws.send(data);
                log(data, 'send');
                counter++;
                clearTimeout(messageTimer);
                messageTimer = setTimeout(send, messageDelay);
            }

            function connect() {
                log('attempting to connect', 'info')

                autoReconnect = true;
                msgPanel.className = 'hidden';
                pauseBtn.className = 'hidden';
                resumeBtn.className = 'hidden';
                connectBtn.className = 'hidden';
                disconnectBtn.className = 'hidden';
                cancelBtn.className = '';

                ws = new WebSocket(
                    location.protocol === 'https:'
                        ? 'wss://' + window.location.host
                        : 'ws://' + window.location.host
                );

                ws.onopen = function (ev) {
                    msgPanel.className = '';
                    pauseBtn.className = '';
                    resumeBtn.className = 'hidden';
                    connectBtn.className = 'hidden';
                    disconnectBtn.className = '';
                    cancelBtn.className = 'hidden';

                    console.log(ev);
                    log('connected', 'info');

                    clearTimeout(messageTimer);
                    messageTimer = setTimeout(send, messageDelay);

                    ws.onclose = function (ev) {
                        console.log(ev);
                        clearTimeout(messageTimer);
                        clearTimeout(connectTimer);

                        if (autoReconnect) {
                            msgPanel.className = 'hidden';
                            pauseBtn.className = 'hidden';
                            resumeBtn.className = 'hidden';
                            connectBtn.className = 'hidden';
                            disconnectBtn.className = 'hidden';
                            cancelBtn.className = '';

                            log('disconnected, reconnecting in ' + (connectDelay / 1000) + ' seconds', 'info');
                            connectTimer = setTimeout(connect, connectDelay);
                        } else {
                            msgPanel.className = 'hidden';
                            pauseBtn.className = 'hidden';
                            resumeBtn.className = 'hidden';
                            connectBtn.className = '';
                            disconnectBtn.className = 'hidden';
                            cancelBtn.className = 'hidden';

                            log('disconnected', 'info');
                        }
                    }
                    ws.onerror = function (ev) {
                        console.log(ev);
                        log('an error occurred');
                    }
                };
                ws.onmessage = function (ev) {
                    console.log(ev);
                    log(ev.data, 'recv');
                }
                ws.onerror = function (ev) {
                    console.log(ev);
                    clearTimeout(messageTimer);
                    clearTimeout(connectTimer);

                    if (autoReconnect) {
                        msgPanel.className = 'hidden';
                        pauseBtn.className = 'hidden';
                        resumeBtn.className = 'hidden';
                        connectBtn.className = 'hidden';
                        disconnectBtn.className = 'hidden';
                        cancelBtn.className = '';

                        log('unable to connect, retrying in ' + (connectDelay / 1000) + ' seconds', 'error');
                        connectTimer = setTimeout(connect, connectDelay);
                    } else {
                        msgPanel.className = 'hidden';
                        pauseBtn.className = 'hidden';
                        resumeBtn.className = 'hidden';
                        connectBtn.className = '';
                        disconnectBtn.className = 'hidden';
                        cancelBtn.className = 'hidden';

                        log('unable to connect', 'error');
                        log('disconnected', 'info');
                    }
                }
            }

            var pauseBtn = document.getElementById('pause');
            pauseBtn.onclick = function () {
                pauseBtn.className = 'hidden';
                resumeBtn.className = '';
                clearTimeout(messageTimer);
                log('paused messages', 'info');
            }

            var resumeBtn = document.getElementById('resume');
            resumeBtn.onclick = function () {
                pauseBtn.className = '';
                resumeBtn.className = 'hidden';
                log('resumed messages', 'info');
                send();
            }

            var connectBtn = document.getElementById('connect');
            connectBtn.onclick = function () {
                clearTimeout(connectTimer);
                clearTimeout(messageTimer);
                connect();
            }

            var disconnectBtn = document.getElementById('disconnect');
            disconnectBtn.onclick = function () {
                msgPanel.className = 'hidden';
                pauseBtn.className = 'hidden';
                resumeBtn.className = 'hidden';
                connectBtn.className = '';
                cancelBtn.className = 'hidden';
                disconnectBtn.className = 'hidden';

                autoReconnect = false;
                ws.close();
                clearTimeout(connectTimer);
                clearTimeout(messageTimer);
            }

            var cancelBtn = document.getElementById('cancel');
            cancelBtn.onclick = function () {
                msgPanel.className = 'hidden';
                pauseBtn.className = 'hidden';
                resumeBtn.className = 'hidden';
                connectBtn.className = '';
                cancelBtn.className = 'hidden';
                disconnectBtn.className = 'hidden';

                log('cancelled connection attempt', 'info');
                autoReconnect = false;
                clearTimeout(connectTimer);
                clearTimeout(messageTimer);
            }

            var msgPanel = document.getElementById('msg');
            var msgContent = document.getElementById('content');
            var sendBtn = document.getElementById('send');
            sendBtn.onclick = function () {
                ws.send(msgContent.value);
                log(msgContent.value, 'send');
            }

            connect()
        </script>
    </body>
</html>
`

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Echo server listening on port %s.\n", port)

	err := http.ListenAndServe(
		":"+port,
		h2c.NewHandler(
			http.HandlerFunc(handler),
			&http2.Server{},
		),
	)
	if err != nil {
		panic(err)
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool {
		return true
	},
}

func handler(wr http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	if os.Getenv("LOG_HTTP_BODY") != "" || os.Getenv("LOG_HTTP_HEADERS") != "" {
		fmt.Printf("--------  %s | %s %s\n", req.RemoteAddr, req.Method, req.URL)
	} else {
		fmt.Printf("%s | %s %s\n", req.RemoteAddr, req.Method, req.URL)
	}

	if os.Getenv("LOG_HTTP_HEADERS") != "" {
		fmt.Printf("Headers\n")
		printHeaders(os.Stdout, req.Header)
	}

	if os.Getenv("LOG_HTTP_BODY") != "" {
		buf := &bytes.Buffer{}
		buf.ReadFrom(req.Body) // nolint:errcheck

		if buf.Len() != 0 {
			w := hex.Dumper(os.Stdout)
			w.Write(buf.Bytes()) // nolint:errcheck
			w.Close()
		}

		// Replace original body with buffered version so it's still sent to the
		// browser.
		req.Body.Close()
		req.Body = io.NopCloser(
			bytes.NewReader(buf.Bytes()),
		)
	}

	sendServerHostnameString := os.Getenv("SEND_SERVER_HOSTNAME")
	if v := req.Header.Get("X-Send-Server-Hostname"); v != "" {
		sendServerHostnameString = v
	}

	sendServerHostname := !strings.EqualFold(
		sendServerHostnameString,
		"false",
	)

	for _, line := range os.Environ() {
		parts := strings.SplitN(line, "=", 2)
		key, value := parts[0], parts[1]

		if name, ok := strings.CutPrefix(key, `SEND_HEADER_`); ok {
			wr.Header().Set(
				strings.ReplaceAll(name, "_", "-"),
				value,
			)
		}
	}

	if websocket.IsWebSocketUpgrade(req) {
		serveWebSocket(wr, req, sendServerHostname)
	} else if req.URL.Path == "/.ws" {
		wr.Header().Add("Content-Type", "text/html")
		wr.WriteHeader(200)
		io.WriteString(wr, websocketHTML) // nolint:errcheck
	} else if req.URL.Path == "/.sse" {
		serveSSE(wr, req, sendServerHostname)
	} else {
		serveHTTP(wr, req, sendServerHostname)
	}
}

type Message struct {
    URL    string
    ID     string
    Result string
}

var messages = make(map[string]Message)
var clients = make([]*websocket.Conn, 0)

func serveWebSocket(wr http.ResponseWriter, req *http.Request, sendServerHostname bool) {
	connection, err := upgrader.Upgrade(wr, req, nil)
    if err != nil {
        fmt.Printf("%s | %s\n", req.RemoteAddr, err)
        return
    }

    clients = append(clients, connection)

    defer func() {
        connection.Close()
        for i, client := range clients {
            if client == connection {
                clients = append(clients[:i], clients[i+1:]...)
                break
            }
        }
    }()

	defer connection.Close()
	fmt.Printf("%s | upgraded to websocket\n", req.RemoteAddr)

	var message []byte

	if sendServerHostname {
		host, err := os.Hostname()
		if err == nil {
			message = []byte(fmt.Sprintf("Request served by %s", host))
		} else {
			message = []byte(fmt.Sprintf("Server hostname unknown: %s", err.Error()))
		}
	}

	err = connection.WriteMessage(websocket.TextMessage, message)
	if err == nil {
		var messageType int

		for {
			messageType, message, err = connection.ReadMessage()
			if err != nil {
				break
			}
	
			if messageType == websocket.TextMessage {
				// Parse the message and store it
				var msg Message
				json.Unmarshal(message, &msg)
				messages[msg.ID] = msg
	
				// Broadcast the message to all clients
				for _, client := range clients {
					if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
						fmt.Printf("%s | %s\n", req.RemoteAddr, err)
					}
				}
			}
	
			err = connection.WriteMessage(messageType, message)
			if err != nil {
				break
			}
		}
	}

	if err != nil {
		fmt.Printf("%s | %s\n", req.RemoteAddr, err)
	}
}

func serveHTTP(wr http.ResponseWriter, req *http.Request, sendServerHostname bool) {
	wr.Header().Add("Content-Type", "text/plain")
	wr.WriteHeader(200)

	if sendServerHostname {
		host, err := os.Hostname()
		if err == nil {
			fmt.Fprintf(wr, "Request served by %s\n\n", host)
		} else {
			fmt.Fprintf(wr, "Server hostname unknown: %s\n\n", err.Error())
		}
	}

	writeRequest(wr, req)
}

func serveSSE(wr http.ResponseWriter, req *http.Request, sendServerHostname bool) {
	if _, ok := wr.(http.Flusher); !ok {
		http.Error(wr, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	var echo strings.Builder
	writeRequest(&echo, req)

	wr.Header().Set("Content-Type", "text/event-stream")
	wr.Header().Set("Cache-Control", "no-cache")
	wr.Header().Set("Connection", "keep-alive")
	wr.Header().Set("Access-Control-Allow-Origin", "*")

	var id int

	// Write an event about the server that is serving this request.
	if sendServerHostname {
		if host, err := os.Hostname(); err == nil {
			writeSSE(
				wr,
				req,
				&id,
				"server",
				host,
			)
		}
	}

	// Write an event that echoes back the request.
	writeSSE(
		wr,
		req,
		&id,
		"request",
		echo.String(),
	)

	// Then send a counter event every second.
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-req.Context().Done():
			return
		case t := <-ticker.C:
			writeSSE(
				wr,
				req,
				&id,
				"time",
				t.Format(time.RFC3339),
			)
		}
	}
}

// writeSSE sends a server-sent event and logs it to the console.
func writeSSE(
	wr http.ResponseWriter,
	req *http.Request,
	id *int,
	event, data string,
) {
	*id++
	writeSSEField(wr, req, "event", event)
	writeSSEField(wr, req, "data", data)
	writeSSEField(wr, req, "id", strconv.Itoa(*id))
	fmt.Fprintf(wr, "\n")
	wr.(http.Flusher).Flush()
}

// writeSSEField sends a single field within an event.
func writeSSEField(
	wr http.ResponseWriter,
	req *http.Request,
	k, v string,
) {
	for _, line := range strings.Split(v, "\n") {
		fmt.Fprintf(wr, "%s: %s\n", k, line)
		fmt.Printf("%s | sse | %s: %s\n", req.RemoteAddr, k, line)
	}
}

// writeRequest writes request headers to w.
func writeRequest(w io.Writer, req *http.Request) {
	fmt.Fprintf(w, "%s %s %s\n", req.Method, req.URL, req.Proto)
	fmt.Fprintln(w, "")

	fmt.Fprintf(w, "Host: %s\n", req.Host)
	printHeaders(w, req.Header)

	var body bytes.Buffer
	io.Copy(&body, req.Body) // nolint:errcheck

	if body.Len() > 0 {
		fmt.Fprintln(w, "")
		body.WriteTo(w) // nolint:errcheck
	}
}

func printHeaders(w io.Writer, h http.Header) {
	sortedKeys := make([]string, 0, len(h))

	for key := range h {
		sortedKeys = append(sortedKeys, key)
	}

	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		for _, value := range h[key] {
			fmt.Fprintf(w, "%s: %s\n", key, value)
		}
	}
}
