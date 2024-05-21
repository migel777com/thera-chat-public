package firebase

import (
	"context"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type FirebaseAuthenticator struct {
	*auth.Client
}

func NewFirebaseAuthenticator(ctx context.Context) (*FirebaseAuthenticator, error) {
	opt := option.WithCredentialsFile("./th3rachat-firebase-adminsdk-a3jhr-b45619ef83.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, err
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}

	return &FirebaseAuthenticator{client}, nil
}
