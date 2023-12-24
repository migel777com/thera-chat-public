package handler

import (
	"chatgpt/api/middleware"
	"chatgpt/auth"
	"chatgpt/models"
	"chatgpt/server"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type UserHandler struct {
	Server *server.Server
}

func NewUserHandler(server *server.Server) *UserHandler {
	return &UserHandler{server}
}

func (u *UserHandler) Init() {
	profile := u.Server.Router.Group("/profile", middleware.Authenticate(u.Server.Cache))
	profile.GET("", u.Profile)
	profile.PATCH("/update", u.Update)
}

// Profile godoc
//
//	@Summary		Get user data
//	@Description	Get user data
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	models.User
//	@Failure		400	{object}	middleware.ErrorResponse
//	@Failure		500	{object}	middleware.ErrorResponse
//	@Router			/profile [get]
func (u *UserHandler) Profile(c *gin.Context) {
	ctx := c.Request.Context()

	cacheUser, ok := c.Get("user")
	if !ok {
		c.AbortWithError(http.StatusUnauthorized, errors.New("not authorized"))
		return
	}

	var filter models.FilterParams
	filter.Filter = fmt.Sprintf(`id = '%v'`, cacheUser.(models.User).Id.String())

	var user models.User
	err := u.Server.Db.Get(ctx, filter, &user)
	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// Update godoc
//
//	@Summary		Update user data
//	@Description	Update user data
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			rq	body		models.User	true	"User data"
//	@Success		200	{object}	models.User
//	@Failure		400	{object}	middleware.ErrorResponse
//	@Failure		500	{object}	middleware.ErrorResponse
//	@Router			/profile/update [patch]
func (u *UserHandler) Update(c *gin.Context) {
	ctx := c.Request.Context()

	user, ok := c.Get("user")
	if !ok {
		c.AbortWithError(http.StatusUnauthorized, errors.New("not authorized"))
		return
	}

	var input models.User
	err := c.ShouldBind(&input)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var filter models.FilterParams
	filter.Filter = fmt.Sprintf(`id = '%v'`, user.(models.User).Id.String())

	err = u.Server.Db.Update(ctx, filter, &input)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var updatedUser models.User
	err = u.Server.Db.Get(ctx, filter, &updatedUser)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	//TODO move through user tokens table and change them
	token, _ := c.Get("token")
	err = u.Server.Cache.SetHash(ctx, auth.RedisAccessPath+token.(string), updatedUser, 24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}
