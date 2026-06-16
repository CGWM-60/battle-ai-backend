package realtime

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func ParkWSHandler(c *gin.Context) { // gin.Context if using gin, otherwise http
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil { return }
	ParkHubInstance.Register(conn)
	defer ParkHubInstance.Unregister(conn)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil { break }
	}
}