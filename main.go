package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os/exec"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(_ *http.Request) bool { return true },
}

type gitter struct {
	name string
}

// processMessage processes the message and returns a
// any error it encounters during the message processing
func (g *gitter) processMessage(msg string) (result []byte, err error) {
	cmd := exec.Command("git", msg)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		fmt.Println(err)
		return nil, err
	}
	return out.Bytes(), nil
}

func (g *gitter) writeMessage(conn *websocket.Conn, msg []byte) error {
	return conn.WriteMessage(websocket.TextMessage, msg)
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
			handleErr(err, " <- reading")
			g.handleError(conn, "error reading the message: "+err.Error())
		}
		fmt.Println(string(p))
		result, err := g.processMessage(string(p))
		if err != nil {
			handleErr(err, " <- processing")
			g.handleError(conn, err.Error())
		}
		err = g.writeMessage(conn, result)
		if err != nil {
			handleErr(err, " <- writing")
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