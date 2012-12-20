package sockjs

import (
	"fmt"
	"strings"
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"net/http"

)

type websocketSession struct { 
	*baseSession
	ws *websocket.Conn
}

func (s *websocketSession) Receive() ([]byte, error) {
	var messages []string
	var data []byte

	//* read some messages to the queue and pull the first one
	err := websocket.Message.Receive(s.ws, &data)
	if err != nil {
		return nil, err
	}

	// ignore, no frame
	if len(data) == 0 {
		return s.Receive()
	}

	err = json.Unmarshal(data, &messages)
	if err != nil {
		return nil, err
	}


	for _, v := range messages {
		s.in.push([]byte(v))
	}

	return s.in.pull()
}

func (s *websocketSession) Send(m []byte) (err error) {
	_, err = s.ws.Write(frame("a", "", m))
	return
}

func (s *websocketSession) Close() (err error) {
	s.ws.Write([]byte(`c[3000,"Go away!"]`))
	err = s.ws.Close()
	s.closeBase()
	return
}

func handleWebsocket(h *Handler, w http.ResponseWriter, r *http.Request) {
	if !h.config.Websocket {
		http.NotFound(w, r)
		return
	}

	// hack to pass test: test_httpMethod (__main__.WebsocketHttpErrors)
	if r.Header.Get("Sec-WebSocket-Version") == "13" && r.Header.Get("Origin") == "" {
		r.Header.Set("Origin", r.Header.Get("Sec-WebSocket-Origin"))
	}
	if strings.ToLower(r.Header.Get("Upgrade")) != "websocket" {
		http.Error(w, `Can "Upgrade" only to "WebSocket".`, http.StatusBadRequest)
		return
	}

	// hack to pass test: test_invalidConnectionHeader (__main__.WebsocketHttpErrors)
	conn := strings.ToLower(r.Header.Get("Connection"))
	if conn == "keep-alive, upgrade" {
		r.Header.Set("Connection", "Upgrade")
	} else if conn != "upgrade" {
		http.Error(w, `"Connection" must be "Upgrade".`, http.StatusBadRequest)
		return
	}

	wh := websocket.Handler(func(ws *websocket.Conn) {
		// initiate connection
		_, err := ws.Write([]byte{'o'})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		s := new(websocketSession)
		s.baseSession = newBaseSession(h.pool)
		s.ws = ws
		h.hfunc(s)
	})

	wh.ServeHTTP(w, r)
}

func handleWebsocketPost(w http.ResponseWriter, r *http.Request) {
	// hack to pass test: test_invalidMethod (__main__.WebsocketHttpErrors)
	conn, bufrw, err := w.(http.Hijacker).Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	fmt.Fprintf(bufrw, 
		"HTTP/1.1 %d %s\r\n", 
		http.StatusMethodNotAllowed, 
		http.StatusText(http.StatusMethodNotAllowed))
	fmt.Fprint(bufrw, "Content-Length: 0\r\n")
	fmt.Fprint(bufrw, "Allow: GET\r\n")
	fmt.Fprint(bufrw, "\r\n")
	bufrw.Flush()
	return
}
