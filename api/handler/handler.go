package handler

import (
	"chatgpt/server"
)

type Handler struct {
	AuthHandler *AuthHandler
	UserHandler *UserHandler
	ChatHandler *ChatHandler
}

func NewHandler(server *server.Server) *Handler {
	return &Handler{
		AuthHandler: NewAuthHandler(server),
		UserHandler: NewUserHandler(server),
		ChatHandler: NewChatHandler(server),
	}
}

func (h *Handler) InitRoutes() {
	h.AuthHandler.Init()
	h.UserHandler.Init()
	h.ChatHandler.Init()
}
