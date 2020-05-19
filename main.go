package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

var roomManager *Manager

func main() {
	roomManager = NewRoomManager()
	router := gin.Default()
	dbInit()
	router.SetHTMLTemplate(html)

	router.GET("/room/:roomid", roomGET)
	router.POST("/room/:roomid", roomPOST)
	router.DELETE("/room/:roomid", roomDELETE)
	router.GET("/stream/:roomid", stream)

	router.Run(":8080")
}

type ChatMessage struct {
	gorm.Model
	UserId string
	RoomId string
	Text   string
}

func dbInit() {
	db := dbOpen()
	db.AutoMigrate(&ChatMessage{})
	defer db.Close()
}

func dbOpen() *gorm.DB {
	db, err := gorm.Open("sqlite3", "test.sqlite3")
	if err != nil {
		panic("Failed to open DB (dbOpen)")
	}
	return db
}

func dbInsert(userid string, roomid string, text string) {
	db := dbOpen()
	db.Create(&ChatMessage{UserId: userid, RoomId: roomid, Text: text})
	defer db.Close()
}

func dbGetAll(roomid string) []ChatMessage {
	db := dbOpen()
	var messages []ChatMessage
	db.Order("created_at").Where("room_id = ?", roomid).Find(&messages)
	db.Close()
	return messages
}

func stream(c *gin.Context) {
	roomid := c.Param("roomid")
	listener := roomManager.OpenListener(roomid)
	defer roomManager.CloseListener(roomid, listener)

	clientGone := c.Writer.CloseNotify()
	c.Stream(func(w io.Writer) bool {
		select {
		case <-clientGone:
			return false
		case message := <-listener:
			c.SSEvent("message", message)
			return true
		}
	})
}

func roomGET(c *gin.Context) {
	roomid := c.Param("roomid")
	userid := fmt.Sprint(rand.Int31())
	chatmessages := dbGetAll(roomid)
	c.HTML(http.StatusOK, "chat_room", gin.H{
		"roomid":   roomid,
		"userid":   userid,
		"messages": chatmessages,
	})
}

func roomPOST(c *gin.Context) {
	roomid := c.Param("roomid")
	userid := c.PostForm("user")
	message := c.PostForm("message")
	roomManager.Submit(userid, roomid, message)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": message,
	})
	dbInsert(userid, roomid, message)
}

func roomDELETE(c *gin.Context) {
	roomid := c.Param("roomid")
	roomManager.DeleteBroadcast(roomid)
}
