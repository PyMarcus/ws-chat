package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/CloudyKit/jet/v6"
	"github.com/gorilla/websocket"
)

var wsChan = make(chan WsPayload)

var clients = make(map[WebSocketConnection]string)

var views = jet.NewSet(
	jet.NewOSFileSystemLoader("./html"),
	jet.InDevelopmentMode(),
)

func Home(w http.ResponseWriter, r *http.Request) {
	err := renderTemplate(w, "home.jet", nil)
	if err != nil {
		log.Println(err)
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string, data jet.VarMap) error {
	view, err := views.GetTemplate(tmpl)

	if err != nil {
		log.Println(err)
		return err
	}

	err = view.Execute(w, data, nil)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type WsJsonResponse struct {
	Action         string   `json:"action"`
	Message        string   `json:"message"`
	MessageType    string   `json:"message_type"`
	ConnectedUsers []string `json:"connected_users"`
}

type WsPayload struct {
	Action   string              `json:"action"`
	Username string              `json:"username"`
	Message  string              `json:"message"`
	Conn     WebSocketConnection `json:"-"`
}

type WebSocketConnection struct {
	*websocket.Conn
}

func WsEndpoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgradeConnection.Upgrade(w, r, nil)

	conn := WebSocketConnection{Conn: ws}
	clients[conn] = ""

	if err != nil {
		log.Println(err)
	}

	var response WsJsonResponse

	response.Message = "<em>Msg</em>"
	ws.WriteJSON(response)

	go ListenForWs(&conn)
}

func ListenForWs(conn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error", fmt.Sprintf("%v", r))
		}
	}()

	var payload WsPayload
	for {
		err := conn.ReadJSON(&payload)

		if err == nil {
			payload.Conn = *conn
			wsChan <- payload
		}
	}
}

func ListenToWsChan() {
	var response WsJsonResponse

	for {
		evt := <-wsChan

		switch evt.Action {
		case "username":
			clients[evt.Conn] = evt.Username
			response.Action = "list_users"
		case "left":
			response.Action = "list_users"
			delete(clients, evt.Conn)
		case "message":
			response.Action = "message"
			response.Message = fmt.Sprintf("<strong>%s</strong>: %s", evt.Username, evt.Message)
		}

		users := getUserList()
		response.ConnectedUsers = users
		broadCastToAll(response)
	}
}

func getUserList() []string {
	var userList []string
	for _, c := range clients {
		if c != "" {
			userList = append(userList, c)
		}
	}
	sort.Strings(userList)
	return userList
}

func broadCastToAll(response WsJsonResponse) {
	for client := range clients {
		log.Printf("Response %v \n", response)
		err := client.WriteJSON(response)
		if err != nil {
			log.Println("Socket error ")
			client.Close()
			delete(clients, client)
		}
	}
}
