package google

import (
	"context"
	"errors"
	"google.golang.org/api/idtoken"
	"strings"
)

type GoogleAuthenticator struct {
	Audiences []string
}

type GoogleUser struct {
	UserId    string
	Email     string
	FirstName string
	LastName  string
}

func (g GoogleAuthenticator) ValidateIdToken(token string) (*GoogleUser, error) {
	payload, err := idtoken.Validate(context.Background(), token, "")
	if err != nil {
		return nil, err
	}

	claims := payload.Claims

	user := &GoogleUser{
		UserId: payload.Subject,
	}

	err = validateAudience(g.Audiences, (claims)["aud"].(string))

	if err != nil {
		return nil, err
	}

	email, ok := (claims)["email"]

	if ok {
		user.Email = strings.TrimSpace(strings.ToLower(email.(string)))
	} else {
		return nil, errors.New("email not found in claims")
	}

	firstName, ok := claims["given_name"]

	if ok {
		user.FirstName = firstName.(string)
	}

	lastName, ok := claims["family_name"]

	if ok {
		user.LastName = lastName.(string)
	}

	return user, nil
}

// Validate that we have a valid audience, using a list since we have different audiences for iOS and Android.
func validateAudience(valid []string, audience string) error {
	for _, s := range valid {
		if s == audience {
			return nil
		}
	}
	return errors.New("audience provided does not match aud claim in the JWT, audience " + audience)
}
