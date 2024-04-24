package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var clients = make(map[string]*websocket.Conn)
var broadcast = make(chan Message)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var mutex sync.Mutex

// Message object
type Message struct {
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Content  string `json:"content"`
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	router := mux.NewRouter()
	router.Use(enableCORS)
	router.HandleFunc("/ws/{username}", handleConnections)
	router.HandleFunc("/users", getUsers)
	go handleMessages()
	http.ListenAndServe(":8080", router)
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}
	defer ws.Close()

	mutex.Lock()
	clients[username] = ws
	fmt.Printf("added user %s\n", username)
	mutex.Unlock()

	broadcastUserList()

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			mutex.Lock()
			delete(clients, username)
			mutex.Unlock()
			break
		}
		broadcast <- msg
	}
}

func broadcastUserList() {
	var userList []string
	for user := range clients {
		userList = append(userList, user)
	}
	userListJSON, _ := json.Marshal(userList)
	for _, client := range clients {
		client.WriteMessage(websocket.TextMessage, userListJSON)
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		mutex.Lock()
		if recipient, ok := clients[msg.Receiver]; ok {
			err := recipient.WriteJSON(msg)
			if err != nil {
				recipient.Close()
				delete(clients, msg.Receiver)
			}
		}
		mutex.Unlock()
	}
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	var userList []string
	for username := range clients {
		userList = append(userList, username)
	}

	json.NewEncoder(w).Encode(userList)
}
