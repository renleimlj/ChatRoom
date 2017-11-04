package main

import (
    "fmt"
    "github.com/go-martini/martini"
    "github.com/gorilla/websocket"
    "log"
    "net/http"
    "text/template"
)

const (
    readBufferSize  = 1024
    writeBufferSize = 1024
)

type Client struct {
    conn *websocket.Conn
    messages chan []byte
}

var clients map[*Client]bool // 存储所有的链接
var broadcast chan []byte    // 广播聊天的chan
var addClients chan *Client  // 新链接进来的chan

func (c *Client) readPump() {
    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
                log.Printf("error: %v", err)
            }
            break
        }
        fmt.Printf("receive message is :%s\n", message)
        broadcast <- message
    }
}

func (c *Client) writePump() {
    for {
        select {
        case message := <-c.messages:
            fmt.Printf("send message is :%s\n", message)
            c.conn.WriteMessage(1, message)
        }
    }
}

func manager() {
    clients = make(map[*Client]bool)
    broadcast = make(chan []byte, 10)
    addClients = make(chan *Client)

    for {
        select {
        case message := <-broadcast:
            for client := range clients {
                select {
                case client.messages <- message:
                default:
                    close(client.messages)
                    delete(clients, client)
                }
            }
        case itemClient := <-addClients:
            clients[itemClient] = true
        }
    }
}

func main() {
    var indexTemplate = template.Must(template.ParseFiles("index.html"))
    m := martini.Classic()
    m.Get("/", func(res http.ResponseWriter, req *http.Request) {
        res.Header().Set("Content-Type", "text/html; charset=utf-8")
        indexTemplate.Execute(res, req.Host)
    })

    m.Get("/ws", func(res http.ResponseWriter, req *http.Request) { // res and req are injected by Martini
        conn, err := websocket.Upgrade(res, req, nil, readBufferSize, writeBufferSize)
        if err != nil {
            log.Println(err)
            return
        }
        client := &Client{conn: conn, messages: make(chan []byte, 5)}
        addClients <- client
        go client.writePump()
        client.readPump()
    })

    go manager()
    m.Run()
}