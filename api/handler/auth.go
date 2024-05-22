package handler

import (
	"chatgpt/auth"
	"chatgpt/models"
	"chatgpt/server"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"net/http"
	"time"
)

type AuthHandler struct {
	Server *server.Server
}

func NewAuthHandler(server *server.Server) *AuthHandler {
	return &AuthHandler{server}
}

func (a *AuthHandler) Init() {
	a.Server.Router.POST("/register", a.Register)
	//a.Server.Router.POST("/verify", a.Verify)
	a.Server.Router.POST("/auth/phone", a.LoginPhone)
	a.Server.Router.POST("/auth/email", a.LoginEmail)
	a.Server.Router.GET("/token/refresh/:token", a.Refresh)

	a.Server.Router.POST("/auth/firebase", a.FirebaseAuth)
}

type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// Register godoc
//
//	@Summary		Register new user
//	@Description	add new user to db and return access and refresh token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			rq	body		models.AuthorizationFields	true	"Input data"
//	@Success		200	{object}	TokenResponse
//	@Failure		400	{object}	models.ErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/register [post]
func (a *AuthHandler) Register(c *gin.Context) {
	ctx := c.Request.Context()

	var input models.AuthorizationFields
	err := c.ShouldBind(&input)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = input.Validate()
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	var filter models.FilterParams
	if len(input.Email) > 0 {
		filter.Filter = fmt.Sprintf(`email = '%v'`, input.Email)
	} else if len(input.Phone) > 0 {
		filter.Filter = fmt.Sprintf(`phone = '%v'`, input.Phone)
	}

	var user models.User
	err = a.Server.Db.Get(ctx, filter, &user)
	if !models.IsErrNotFound(err) && len(user.Password) > 0 {
		c.AbortWithError(http.StatusBadRequest, errors.New("user already registered"))
		return
	} else if models.AllowErrNotFound(err) != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if input.Password != input.RePassword {
		c.AbortWithError(http.StatusBadRequest, errors.New("passwords not same"))
		return
	}

	thread, err := a.Server.AI.NewThread(ctx)
	if err != nil {
		c.Error(err)
	}

	user = models.User{
		Phone:    input.Phone,
		Password: input.Password,
		Email:    input.Email,
		Name:     input.Name,
		Surname:  input.Surname,
		Thread:   thread.ID,
	}
	err = a.Server.Db.Create(ctx, &user)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	access, refresh, err := auth.GetAuthTokens(user.Id.String(), a.Server.Configuration.SecretKeyAccess, a.Server.Configuration.SecretKeyRefresh)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = a.Server.Cache.SetHash(ctx, auth.RedisAccessPath+access.Plaintext, user, 24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = a.Server.Cache.SetHash(ctx, auth.RedisRefreshPath+refresh.Plaintext, user, 7*24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = a.Server.Cache.SetHash(ctx, RedisThread+user.Id.String(), thread.ID, 24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, TokenResponse{access.Plaintext, refresh.Plaintext})
}

// LoginPhone godoc
//
//	@Summary		Login by phone number
//	@Description	Login by phone number
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			rq	body		models.AuthorizationFields	true	"Fill in only phone and password"
//	@Success		200	{object}	TokenResponse
//	@Failure		400	{object}	models.AdvancedErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/auth/phone [post]
func (a *AuthHandler) LoginPhone(c *gin.Context) {
	ctx := c.Request.Context()

	var input models.AuthorizationFields
	err := c.ShouldBind(&input)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var filter models.FilterParams
	if len(input.Phone) > 0 {
		filter.Filter = fmt.Sprintf(`
			phone    = '%v' and
			password = '%v'`,
			input.Phone, input.Password)
	} else {
		c.AbortWithError(http.StatusBadRequest, models.AdvancedErrorResponse{
			Key:     "phone_field",
			Code:    http.StatusBadRequest,
			Message: "Поле 'phone' должно быть заполнено.",
		})
		return
	}

	var user models.User
	err = a.Server.Db.Get(ctx, filter, &user)
	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, models.AdvancedErrorResponse{
			Key:     "auth_fields",
			Code:    http.StatusUnauthorized,
			Message: "Неверный логин или пароль.",
		})
		return
	}

	access, refresh, err := auth.GetAuthTokens(user.Id.String(), a.Server.Configuration.SecretKeyAccess, a.Server.Configuration.SecretKeyRefresh)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// TODO User tokens table
	//err = a.Server.Db.Create(ctx, &user)
	//if err != nil {
	//	c.AbortWithError(http.StatusInternalServerError, err)
	//	return
	//}

	err = a.Server.Cache.SetHash(ctx, auth.RedisAccessPath+access.Plaintext, user, 24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = a.Server.Cache.SetHash(ctx, auth.RedisRefreshPath+refresh.Plaintext, user, 7*24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, TokenResponse{access.Plaintext, refresh.Plaintext})
}

// LoginEmail godoc
//
//	@Summary		Login by email
//	@Description	Login by email
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			rq	body		models.AuthorizationFields	true	"Fill in only email and password"
//	@Success		200	{object}	TokenResponse
//	@Failure		400	{object}	models.AdvancedErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/auth/email [post]
func (a *AuthHandler) LoginEmail(c *gin.Context) {
	ctx := c.Request.Context()

	var input models.AuthorizationFields
	err := c.ShouldBind(&input)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var filter models.FilterParams
	if len(input.Email) > 0 {
		filter.Filter = fmt.Sprintf(`
			email    = '%v' and
			password = '%v'`,
			input.Email, input.Password)
	} else {
		c.AbortWithError(http.StatusBadRequest, models.AdvancedErrorResponse{
			Key:     "email_field",
			Code:    http.StatusBadRequest,
			Message: "Поле 'email' должно быть заполнено.",
		})
		return
	}

	var user models.User
	err = a.Server.Db.Get(ctx, filter, &user)
	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, models.AdvancedErrorResponse{
			Key:     "auth_fields",
			Code:    http.StatusUnauthorized,
			Message: "Неверный логин или пароль.",
		})
		return
	}

	access, refresh, err := auth.GetAuthTokens(user.Id.String(), a.Server.Configuration.SecretKeyAccess, a.Server.Configuration.SecretKeyRefresh)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// TODO User tokens table
	//err = a.Server.Db.Create(ctx, &user)
	//if err != nil {
	//	c.AbortWithError(http.StatusInternalServerError, err)
	//	return
	//}

	err = a.Server.Cache.SetHash(ctx, auth.RedisAccessPath+access.Plaintext, user, 24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = a.Server.Cache.SetHash(ctx, auth.RedisRefreshPath+refresh.Plaintext, user, 7*24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, TokenResponse{access.Plaintext, refresh.Plaintext})
}

// FirebaseAuth godoc
//
//	@Summary		Register new user
//	@Description	add new user to db and return access and refresh token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			rq	body		models.FirebaseAuthFields	true	"Input data"
//	@Success		200	{object}	TokenResponse
//	@Failure		400	{object}	models.AdvancedErrorResponse
//	@Failure		500	{object}	models.ErrorResponse
//	@Router			/auth/firebase [post]
func (a *AuthHandler) FirebaseAuth(c *gin.Context) {
	ctx := c.Request.Context()

	var input models.FirebaseAuthFields
	err := c.ShouldBind(&input)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	firebaseUser, err := a.Server.Firebase.GetUser(ctx, input.UserUID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	provider := firebaseUser.ProviderUserInfo[0].ProviderID

	var filter models.FilterParams
	filter.Filter = fmt.Sprintf(`email = '%v'`, firebaseUser.Email)

	var thread openai.Thread
	var user models.User
	err = a.Server.Db.Get(ctx, filter, &user)
	if models.IsErrNotFound(err) {
		thread, err = a.Server.AI.NewThread(ctx)
		if err != nil {
			c.Error(err)
		}

		user = models.User{
			Email:  firebaseUser.Email,
			Name:   firebaseUser.UserInfo.DisplayName,
			Thread: thread.ID,
		}

		switch provider {
		case "google.com":
			user.IsGoogle = true
		case "apple.com":
			user.IsApple = true
		}

		err = a.Server.Db.Create(ctx, &user)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	} else if models.AllowErrNotFound(err) != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	} else if (provider == "google.com" && !user.IsGoogle) || (provider == "apple.com" && !user.IsApple) {
		switch provider {
		case "google.com":
			user.IsGoogle = true
		case "apple.com":
			user.IsApple = true
		}

		err = a.Server.Db.Update(ctx, filter, &user)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	access, refresh, err := auth.GetAuthTokens(user.Id.String(), a.Server.Configuration.SecretKeyAccess, a.Server.Configuration.SecretKeyRefresh)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = a.Server.Cache.SetHash(ctx, auth.RedisAccessPath+access.Plaintext, user, 24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = a.Server.Cache.SetHash(ctx, auth.RedisRefreshPath+refresh.Plaintext, user, 7*24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = a.Server.Cache.SetHash(ctx, RedisThread+user.Id.String(), thread.ID, 24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, TokenResponse{access.Plaintext, refresh.Plaintext})
}

// Refresh godoc
//
//	@Summary		Refresh tokens
//	@Description	creates new access and refresh tokens
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			refreshToken	path		string	true	"Refresh Token"
//	@Success		200				{object}	TokenResponse
//	@Failure		400				{object}	models.AdvancedErrorResponse
//	@Failure		500				{object}	models.ErrorResponse
//	@Router			/token/refresh/:refreshToken [get]
func (a *AuthHandler) Refresh(c *gin.Context) {
	ctx := c.Request.Context()

	token := c.Param("token")

	var user models.User
	err := a.Server.Cache.GetHash(ctx, auth.RedisRefreshPath+token, &user)
	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, err)
		return
	}

	var filter models.FilterParams
	filter.Filter = fmt.Sprintf(`id = '%v'`, user.Id.String())

	err = a.Server.Db.Get(ctx, filter, &user)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	access, refresh, err := auth.GetAuthTokens(user.Id.String(), a.Server.Configuration.SecretKeyAccess, a.Server.Configuration.SecretKeyRefresh)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// TODO move through user tokens table and change them
	// TODO User tokens table
	//err = a.Server.Db.Create(ctx, &user)
	//if err != nil {
	//	c.AbortWithError(http.StatusInternalServerError, err)
	//	return
	//}

	err = a.Server.Cache.SetHash(ctx, auth.RedisAccessPath+access.Plaintext, user, 24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = a.Server.Cache.SetHash(ctx, auth.RedisRefreshPath+refresh.Plaintext, user, 7*24*time.Hour)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, TokenResponse{access.Plaintext, refresh.Plaintext})
}
