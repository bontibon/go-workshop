package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/bontibon/refresh-go-workshop/snakes"
	"github.com/gorilla/websocket"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP address to listen on")
	flag.Parse()

	server := snakes.NewServer()
	go server.Run()

	mux := http.NewServeMux()

	mux.HandleFunc("/viewer", func(w http.ResponseWriter, r *http.Request) {
		viewer, err := ioutil.ReadFile("viewer.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Write(viewer)
	})

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	mux.HandleFunc("/viewer/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer conn.Close()

		defer log.Printf("Viewer disconnected (%s)", conn.RemoteAddr())

		log.Printf("Viewer connected (%s)", conn.RemoteAddr())

		client, err := snakes.NewViewerConn(conn)
		if err != nil {
			log.Printf("could not create conn: %s", err)
			return
		}
		if err := server.AddViewer(client); err != nil {
			log.Printf("could not add client: %s", err)
			return
		}
		defer server.RemoveViewer(client)
		if err := client.Run(); err != nil {
			log.Printf("Client error (%s): %s", conn.RemoteAddr(), err)
		}
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer conn.Close()
		defer log.Printf("Client disconnected (%s)", conn.RemoteAddr())

		log.Printf("Client connected (%s)", conn.RemoteAddr())

		client, err := snakes.NewServerConn(conn, r)
		if err != nil {
			log.Printf("could not create conn: %s", err)
			return
		}
		if err := server.AddClient(client); err != nil {
			log.Printf("could not add client: %s", err)
			return
		}
		defer server.RemoveClient(client)
		if err := client.Run(); err != nil {
			log.Printf("Client error (%s): %s", conn.RemoteAddr(), err)
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-type", "text/html; charset=utf-8")
		io.WriteString(w, `<h1>Refresh AV: SNAKES <span style="font-weight: normal">🐍</span></h1><ul><li><a href="/viewer">/viewer</a></li><li><a href="/ws">/ws</a> (client endpoint)</li></ul>`)
	})

	log.Printf("Starting server on %s\n", *addr)
	if err := http.ListenAndServe(*addr, mux); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
