//go:generate go-bindata-assetfs public/...
package main

import (
	"flag"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/rumlang/playground/forms"
	"github.com/satori/go.uuid"
	"github.com/trumae/valente"
	"github.com/trumae/valente/status"

	"github.com/gorilla/websocket"
)

//App is a Web Application representation
type App struct {
	valente.App
}

const timeout = 300
const gctime = 1

var (
	sessions map[string]*App
	mutex    sync.Mutex

	upgrader = websocket.Upgrader{}
)

//addSession include a new app on sessions
func addSession(key string, app *App) {
	mutex.Lock()
	defer mutex.Unlock()

	sessions[key] = app
}

//getSession return the app by key
func getSession(key string) *App {
	return sessions[key]
}

func gcStepSession() {
	mutex.Lock()
	defer mutex.Unlock()

	now := time.Now().Unix()
	for key, app := range sessions {
		if now-app.LastAccess.Unix() > timeout {
			log.Println("Collecting", key)
			delete(sessions, key)
		}
	}
}

//Initialize inits the App
func (app *App) Initialize() {
	log.Println("App Initialize")

	app.AddForm("home", forms.RumReplForm{})

	app.GoTo("home", nil)
}

func main() {
	port := "8000"
	flag.StringVar(&port, "port", "8000", "http port")

	flag.Parse()

	log.Println("Init sessions")
	sessions = make(map[string]*App)
	mutex = sync.Mutex{}

	go func() {
		for {
			time.Sleep(gctime * time.Second)
			gcStepSession()
		}
	}()

	fs := http.FileServer(assetFS())
	http.Handle("/", fs)

	http.HandleFunc("/status", status.ValenteStatusHandler)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer ws.Close()

		err = ws.WriteMessage(websocket.TextMessage, []byte("__GETSESSION__"))
		if err != nil {
			log.Println(err)
			return
		}

		_, bid, err := ws.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		idSession := string(bid)

		var app *App
		app = getSession(idSession)
		if app == nil {
			su1, err := uuid.NewV4()
			if err != nil {
				log.Println(err)
				return
			}
			u1 := su1.String()
			log.Println("New session", u1)
			app = &App{}
			app.LastAccess = time.Now()
			addSession(u1, app)
			err = ws.WriteMessage(websocket.TextMessage, []byte(u1))
			if err != nil {
				log.Println(err)
				return
			}
			app.WebSocket(ws)
			app.Initialize()
		} else {
			app.LastAccess = time.Now()
			log.Println("Reusing session", idSession)
			err := ws.WriteMessage(websocket.TextMessage, []byte(idSession))
			if err != nil {
				log.Println(err)
				return
			}
			app.WebSocket(ws)
		}
		app.Run()
	})

	log.Println("Server running on port", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Println(err)
	}
}
