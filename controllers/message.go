package controllers

import (
	"net/http"

	"github.com/dartt0n/realtime-chat-backend/forms"
	"github.com/dartt0n/realtime-chat-backend/service"
	"github.com/gin-gonic/gin"
)

type MessageController struct {
	auth   *service.AuthService
	tinode *service.TinodeService
}

var msgForm = new(forms.MessageForm)

func NewMessageController(tinode *service.TinodeService, auth *service.AuthService) *MessageController {
	return &MessageController{tinode: tinode, auth: auth}
}

func (ctrl MessageController) FetchLast(c *gin.Context) {
	_, err := ctrl.auth.ExtractTokenMetadata(c.Request)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "User not logged in"})
		return
	}

	lastMsg, err := ctrl.tinode.FetchLastMsgs()
	if err != nil {
		c.JSON(http.StatusNotAcceptable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, lastMsg)
}

func (ctrl MessageController) SendMsg(c *gin.Context) {
	au, err := ctrl.auth.ExtractTokenMetadata(c.Request)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "User not logged in"})
		return
	}

	var textForm forms.TextMessage
	if err := c.ShouldBind(&textForm); err != nil {
		message := msgForm.Text(err)
		c.JSON(http.StatusNotAcceptable, gin.H{"error": message})
		return
	}

	err = ctrl.tinode.SendMessage(au.AccessUUID, textForm.Content)
	if err != nil {
		c.JSON(http.StatusNotAcceptable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
}
