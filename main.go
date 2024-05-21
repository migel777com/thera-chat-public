package main

import (
	a "chatgpt/ai"
	h "chatgpt/api/handler"
	f "chatgpt/auth/firebase"
	"chatgpt/config"
	"chatgpt/models"
	s "chatgpt/server"
	"chatgpt/store"
	"context"
	"gopkg.in/tylerb/graceful.v1"
	"strconv"

	_ "chatgpt/docs"
)

//	@title			TheraChat API
//	@version		1.0
//	@description	This is a server for communication with ChatGPT.

//	@host	http://64.226.106.122:8080

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configuration := config.NewConfiguration()

	var db models.DbClient
	err := store.NewDB(configuration, &db)
	if err != nil {
		panic(err)
	}
	defer db.CloseClient()

	var cache models.CacheClient
	err = store.NewCacheClient(ctx, configuration, &cache)
	if err != nil {
		panic(err)
	}
	defer cache.CloseClient()

	ai := a.NewAI(configuration)

	firebase, err := f.NewFirebaseAuthenticator(ctx)
	if err != nil {
		panic(err)
	}

	server := s.NewApiServer(configuration, db, cache, ai, firebase)
	server.Init(ctx)

	handler := h.NewHandler(server)
	handler.InitRoutes()

	graceful.Run(":"+strconv.Itoa(server.Configuration.Port), 0, server.Router)
}
