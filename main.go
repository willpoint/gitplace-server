package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(_ *http.Request) bool { return true },
}

// commandType is a constant definition of allowed command types
type commandType string

const (
	version commandType = "version"
	history             = "history"
	diff                = "diff"
	branch              = "branch"
)

type gitter struct{ name string }

// The Result type is composed of the type and
// data produced by processMessage()
type Result struct {
	Typ  string
	Data string
}

// JSON marshals returns the JSON value of result
func (r Result) JSON() ([]byte, error) {
	return json.Marshal(r)
}

// processMessage processes the message and returns a
// any error it encounters during the message processing
func (g *gitter) processMessage(msgs ...string) (result []byte, err error) {
	//the first item in msgs is the type of message
	var commands []string
	if len(msgs) >= 1 {
		commands = msgs[1:]
	}
	cmd := exec.Command("git", commands...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return nil, err
	}
	res := Result{msgs[0], out.String()}
	return res.JSON()
}

func (g *gitter) writeMessage(conn *websocket.Conn, msg []byte) error {
	return conn.WriteJSON(msg)
}

func (g *gitter) handleError(conn *websocket.Conn, msg string) {
	conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

func (g *gitter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		handleErr(err)
	}

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			g.handleError(conn, "error reading the message: "+err.Error())
		}
		msgs := strings.Fields(string(p))
		result, err := g.processMessage(msgs...)
		if err != nil {
			g.handleError(conn, err.Error())
		}
		err = g.writeMessage(conn, result)
		if err != nil {
			g.handleError(conn, err.Error())
		}
	}

}

// This example demonstrates a trivial echo server.
func main() {
	http.Handle("/echo", &gitter{"gitPlace"})
	err := http.ListenAndServe(":12345", nil)
	handleErr(err)
}

func handleErr(err error, msg ...string) {
	if err != nil {
		log.SetPrefix("gitPlace: ")
		log.SetFlags(log.Lshortfile | log.LUTC)
		log.Println(err, msg)
	}
}
