package handler

import (
	"chatgpt/api/middleware"
	"chatgpt/models"
	"chatgpt/server"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"slices"
	"strings"
	"time"
)

const (
	RedisThread = "thread/"
)

type ChatHandler struct {
	Server *server.Server
}

func NewChatHandler(server *server.Server) *ChatHandler {
	return &ChatHandler{server}
}

func (ch *ChatHandler) Init() {
	chat := ch.Server.Router.Group("/chat")
	auth := chat.Group("", middleware.Authenticate(ch.Server.Cache))
	auth.POST("/start", ch.StartChat)
	auth.POST("/message", ch.WriteChatMessage)
	auth.GET("/messages", ch.GetChatMessages)

	anon := chat.Group("/anon")
	anon.POST("/start", ch.StartAnonChat)
	anon.POST("/:id/message", ch.WriteAnonChatMessage)
	anon.GET("/:id/messages", ch.GetAnonChatMessages)
}

// StartChat godoc
//
//	@Summary		Start new chat
//	@Description	starts chat with ChatGPT with authorized user
//	@Tags			chat
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	Response
//	@Failure		400	{object}	models.AdvancedErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/chat/start [post]
func (ch *ChatHandler) StartChat(c *gin.Context) {
	ctx := c.Request.Context()

	cacheUser, ok := c.Get("user")
	if !ok {
		c.AbortWithError(http.StatusUnauthorized, errors.New("not authorized"))
		return
	}

	thread, err := ch.Server.AI.NewThread(ctx)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var filter models.FilterParams
	filter.Filter = fmt.Sprintf(`id = '%v'`, cacheUser.(models.User).Id.String())

	user := models.User{Thread: thread.ID}
	err = ch.Server.Db.Update(ctx, filter, &user)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = ch.Server.Cache.SetHash(ctx, RedisThread+cacheUser.(models.User).Id.String(), thread.ID, 24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, Response{"chat started"})
}

// WriteChatMessage godoc
//
//	@Summary		Writes message
//	@Description	write message from authorized user to the bot and get response
//	@Tags			chat
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			rq	body		models.Message.Text	true	"Message text"
//	@Success		200	{object}	models.Message		"Response from the bot"
//	@Failure		400	{object}	models.AdvancedErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/chat/message [post]
func (ch *ChatHandler) WriteChatMessage(c *gin.Context) {
	ctx := c.Request.Context()

	cacheUser, ok := c.Get("user")
	if !ok {
		c.AbortWithError(http.StatusUnauthorized, errors.New("not authorized"))
		return
	}

	var input models.Message
	err := c.ShouldBind(&input)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if input.Text == "" {
		c.AbortWithError(http.StatusBadRequest, models.AdvancedErrorResponse{
			Key:     "text_field",
			Code:    http.StatusBadRequest,
			Message: "Поле 'text' должно быть заполнено.",
		})
		return
	}

	var threadId string
	err = ch.Server.Cache.GetHash(ctx, RedisThread+cacheUser.(models.User).Id.String(), &threadId)
	if models.IsErrNotFound(err) {
		var filter models.FilterParams
		filter.Filter = fmt.Sprintf(`id = '%v'`, cacheUser.(models.User).Id.String())

		var user models.User
		err = ch.Server.Db.Get(ctx, filter, &user)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if user.Thread == "" {
			user, err = ch.newUserThread(ctx, cacheUser.(models.User))
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		}

		threadId = user.Thread
		err = ch.Server.Cache.SetHash(ctx, RedisThread+cacheUser.(models.User).Id.String(), threadId, 24*time.Hour)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	} else if models.AllowErrNotFound(err) != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	resp, err := ch.Server.AI.NewMessage(ctx, threadId, input.Text)
	if err != nil && strings.Contains(err.Error(), "error, status code: 404, message: No thread found with id") {
		user, err := ch.newUserThread(ctx, cacheUser.(models.User))
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		err = ch.Server.Cache.SetHash(ctx, RedisThread+cacheUser.(models.User).Id.String(), user.Thread, 24*time.Hour)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		resp, err = ch.Server.AI.NewMessage(ctx, user.Thread, input.Text)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	} else if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, models.Message{Text: resp})
}

func (ch *ChatHandler) newUserThread(ctx context.Context, cacheUser models.User) (models.User, error) {
	thread, err := ch.Server.AI.NewThread(ctx)
	if err != nil {
		return models.User{}, err
	}

	var filter models.FilterParams
	filter.Filter = fmt.Sprintf(`id = '%v'`, cacheUser.Id.String())

	user := models.User{Thread: thread.ID}
	err = ch.Server.Db.Update(ctx, filter, &user)
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

// GetChatMessages godoc
//
//	@Summary		Get conversation messages
//	@Description	get messages between bot and authorized user
//	@Tags			chat
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	[]models.Message
//	@Failure		400	{object}	models.AdvancedErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/chat/messages [post]
func (ch *ChatHandler) GetChatMessages(c *gin.Context) {
	ctx := c.Request.Context()

	cacheUser, ok := c.Get("user")
	if !ok {
		c.AbortWithError(http.StatusUnauthorized, errors.New("not authorized"))
		return
	}

	var threadId string
	err := ch.Server.Cache.GetHash(ctx, RedisThread+cacheUser.(models.User).Id.String(), &threadId)
	if models.IsErrNotFound(err) {
		var filter models.FilterParams
		filter.Filter = fmt.Sprintf(`id = '%v'`, cacheUser.(models.User).Id.String())

		var user models.User
		err = ch.Server.Db.Get(ctx, filter, &user)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if user.Thread == "" {
			thread, err := ch.Server.AI.NewThread(ctx)
			if err != nil {
				c.Error(err)
			}

			filter.Filter = fmt.Sprintf(`id = '%v'`, cacheUser.(models.User).Id.String())

			user = models.User{Thread: thread.ID}
			err = ch.Server.Db.Update(ctx, filter, &user)
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		}

		threadId = user.Thread
		err = ch.Server.Cache.SetHash(ctx, RedisThread+cacheUser.(models.User).Id.String(), threadId, 24*time.Hour)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	} else if models.AllowErrNotFound(err) != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	messages, err := ch.Server.AI.GetMessages(ctx, threadId)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	slices.Reverse(messages)

	c.JSON(http.StatusOK, messages)
}

type StartAnonChatResponse struct {
	Id string `json:"id"`
}

// StartAnonChat godoc
//
//	@Summary		Start new anon chat
//	@Description	starts chat with ChatGPT with unauthorized user
//	@Tags			chat
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	StartAnonChatResponse	"ID of anonymous conversation"
//	@Failure		400	{object}	models.AdvancedErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/chat/anon/start [post]
func (ch *ChatHandler) StartAnonChat(c *gin.Context) {
	ctx := c.Request.Context()

	thread, err := ch.Server.AI.NewThread(ctx)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	id, err := uuid.NewUUID()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = ch.Server.Cache.SetHash(ctx, RedisThread+id.String(), thread.ID, 24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, StartAnonChatResponse{id.String()})
}

// WriteAnonChatMessage godoc
//
//	@Summary		Writes message to anon chat
//	@Description	write message from unauthorized user to the bot and get response
//	@Tags			chat
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string				true	"ID of anonymous conversation"
//	@Param			rq	body		models.Message.Text	true	"Message text"
//	@Success		200	{object}	models.Message		"Response from the bot"
//	@Failure		400	{object}	models.AdvancedErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/chat/anon/:id/message [post]
func (ch *ChatHandler) WriteAnonChatMessage(c *gin.Context) {
	ctx := c.Request.Context()

	id := c.Param("id")

	var input models.Message
	err := c.ShouldBind(&input)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if input.Text == "" {
		c.AbortWithError(http.StatusBadRequest, models.AdvancedErrorResponse{
			Key:     "text_field",
			Code:    http.StatusBadRequest,
			Message: "Поле 'text' должно быть заполнено.",
		})
		return
	}

	var thread string
	err = ch.Server.Cache.GetHash(ctx, RedisThread+id, &thread)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	resp, err := ch.Server.AI.NewMessage(ctx, thread, input.Text)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, models.Message{Text: resp})
}

// GetAnonChatMessages godoc
//
//	@Summary		Get anon conversation messages
//	@Description	get messages between bot and unauthorized user
//	@Tags			chat
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"ID of anonymous conversation"
//	@Success		200	{object}	[]models.Message
//	@Failure		400	{object}	models.AdvancedErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/chat/anon/:id/messages [get]
func (ch *ChatHandler) GetAnonChatMessages(c *gin.Context) {
	ctx := c.Request.Context()

	id := c.Param("id")

	var thread string
	err := ch.Server.Cache.GetHash(ctx, RedisThread+id, &thread)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	messages, err := ch.Server.AI.GetMessages(ctx, thread)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	slices.Reverse(messages)

	c.JSON(http.StatusOK, messages)
}
