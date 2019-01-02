package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gorilla/websocket"
)

const (
	headerContentType     = "Content-Type"
	headerApplicationJSON = "application/json"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(_ *http.Request) bool { return true },
}

// The Gitter struct
type Gitter struct {
	Name   string
	Logger *log.Logger
}

// The Result type is composed of the type and
// data produced by processMessage()
type Result struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// ReqBody from client
type ReqBody struct {
	Type string `json:"type"`
	Body string `json:"body"`
}

// NewGitter sets up Gitter with the default
func NewGitter(name string, logger *log.Logger) *Gitter {
	logger.SetFlags(log.LUTC)
	logger.SetPrefix(name + ": ")
	return &Gitter{name, logger}
}

// processMessage processes the message and returns a
// any error it encounters during the message processing
func (g *Gitter) processMessage(msgs ...string) (result interface{}, err error) {

	var commands []string
	if len(msgs) >= 1 {
		commands = msgs[1:]
	}
	cmd := exec.Command("git", commands...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, exitErr
		}
		return nil, err
	}
	res := Result{msgs[0], string(output)}
	return res, nil
}

func (g *Gitter) writeMessage(conn *websocket.Conn, msg interface{}) error {
	return conn.WriteJSON(msg)
}

func (g *Gitter) handleError(conn *websocket.Conn, msg string) {
	res := Result{"error", msg}
	conn.WriteJSON(res)
}

func (g *Gitter) handleHTTPError(w http.ResponseWriter, msg string) {
	res := Result{"error", msg}
	json, err := json.Marshal(res)
	if err != nil {
		w.Header().Set(headerContentType, headerApplicationJSON)
		w.Write([]byte(`"type": "error", "body": "an unexpected error occured!"`))
		return
	}
	w.Write(json)
}

func (g *Gitter) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if !websocket.IsWebSocketUpgrade(r) {
		rb := ReqBody{}
		err := json.NewDecoder(r.Body).Decode(&rb)
		if err != nil {
			g.handleHTTPError(w, err.Error())
			return
		}
		command := rb.Type + " " + rb.Body
		res, err := g.processMessage(strings.Fields(command)...)

		if err != nil {
			fmt.Println(err)
			g.handleHTTPError(w, err.Error())
			return
		}
		rr, err := json.Marshal(res)
		if err != nil {
			g.handleHTTPError(w, err.Error())
			return
		}
		w.Header().Set(headerContentType, headerApplicationJSON)
		w.Write(rr)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		g.handleError(conn, err.Error())
		return
	}

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			g.handleError(conn, "error reading the message: "+err.Error())
			return
		}
		msgs := strings.Fields(string(p))
		result, err := g.processMessage(msgs...)
		if err != nil {
			g.handleError(conn, err.Error())
			return
		}
		err = g.writeMessage(conn, result)
		if err != nil {
			g.handleError(conn, err.Error())
			return
		}
	}

}

func main() {

	// todo(uz) - use flag for logging decision
	url := flag.String("URL", ":5959", "The server URL")
	flag.Parse()

	// Logger
	logger := &log.Logger{}

	// ServeMux
	mux := http.NewServeMux()

	// serve static assets
	// requires esc by github.com/mjibson/esc
	mux.Handle("/", http.FileServer(FS(false)))
	mux.Handle("/_nuxt/", http.FileServer(FS(false)))

	// httpUpgrade Handler
	mux.Handle("/echo", NewGitter("httpUpgrade", logger))

	// http-CORSEnabled Handler
	mux.Handle("/command", AllowCors(CorsConfig{
		AllowHeaders: []string{headerContentType},
	})(NewGitter("http", logger)))

	server := http.Server{
		Addr:    *url,
		Handler: mux,
	}

	// start server
	handleErr(server.ListenAndServe())

}

func handleErr(err error, msg ...string) {
	if err != nil {
		log.SetPrefix("gitPlace: ")
		log.SetFlags(log.Lshortfile | log.LUTC)
		log.Println("an error occured: ", err, msg)
	}
}
