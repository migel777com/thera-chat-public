package apple

import (
	"chatgpt/config"
	"context"
	"fmt"
	"github.com/Timothylock/go-signin-with-apple/apple"
	"strings"
)

type AppleAuthenticator struct {
	AndroidClientId string
	ClientId        string
	PrivateKey      string
	TeamId          string
	KeyId           string
}

type AuthenticatedAppleUser struct {
	AppleUserId string
	Email       string
}

func NewAppleAuth(config *config.Config) *AppleAuthenticator {
	return &AppleAuthenticator{
		AndroidClientId: config.AppleAuthAndroidClientId,
		ClientId:        config.AppleAuthClientId,
		PrivateKey:      config.AppleAuthPrivateKey,
		TeamId:          config.AppleAuthTeamId,
		KeyId:           config.AppleAuthKeyId,
	}
}

func (a AppleAuthenticator) ValidateAuthorizationToken(token string, isAndroid bool) (*AuthenticatedAppleUser, error) {
	var clientId string

	if isAndroid {
		clientId = a.AndroidClientId
	} else {
		clientId = a.ClientId
	}

	secret, err := apple.GenerateClientSecret(a.PrivateKey, a.TeamId, clientId, a.KeyId)

	if err != nil {
		return nil, err
	}

	client := apple.New()

	req := apple.AppValidationTokenRequest{
		ClientID:     clientId,
		ClientSecret: secret,
		Code:         token,
	}

	var resp apple.ValidationResponse

	// Do the verification
	err = client.VerifyAppToken(context.Background(), req, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		fmt.Printf("apple returned an error: %s - %s\n", resp.Error, resp.ErrorDescription)
		if err != nil {
			return nil, err
		}
	}

	// Get the unique user ID
	userId, err := apple.GetUniqueID(resp.IDToken)
	if err != nil {
		return nil, err
	}

	// Get the email
	claim, err := apple.GetClaims(resp.IDToken)
	if err != nil {
		return nil, err
	}

	email := (*claim)["email"].(string)

	return &AuthenticatedAppleUser{
		AppleUserId: userId,
		Email:       strings.TrimSpace(strings.ToLower(email)),
	}, nil
}
