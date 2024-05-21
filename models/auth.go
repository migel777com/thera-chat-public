package models

import (
	"net/http"
	"net/mail"
)

type AuthorizationFields struct {
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Password   string `json:"password"`
	RePassword string `json:"rePassword"`
	Name       string `json:"name"`
	Surname    string `json:"surname"`
}

func (a *AuthorizationFields) Validate() error {
	_, err := mail.ParseAddress(a.Email)
	if err != nil {
		return AdvancedErrorResponse{
			Key:     "email_field",
			Code:    http.StatusBadRequest,
			Message: "Поле 'email' должно содержать действительный адрес электронной почты.",
		}
	}

	if a.Password != a.RePassword {
		return AdvancedErrorResponse{
			Key:     "password_field",
			Code:    http.StatusBadRequest,
			Message: "Поля 'password' и 'rePassword' должны быть одинаковыми.",
		}
	}

	if a.Name == "" {
		return AdvancedErrorResponse{
			Key:     "name_field",
			Code:    http.StatusBadRequest,
			Message: "Поле 'name' должно быть заполнено.",
		}
	}

	if a.Surname == "" {
		return AdvancedErrorResponse{
			Key:     "surname_field",
			Code:    http.StatusBadRequest,
			Message: "Поле 'surname' должно быть заполнено.",
		}
	}

	return nil
}

type FirebaseAuthFields struct {
	UserUID string `json:"userUID"`
}
