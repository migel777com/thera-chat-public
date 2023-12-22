package auth

import (
	"chatgpt/models"
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base32"
	"errors"
	"time"
)

// TODO refactor auth package

// Scope of token usage.
const (
	ScopeAccess      = "access"
	ScopeRefresh     = "refresh"
	RedisAccessPath  = "access/"
	RedisRefreshPath = "refresh/"
	RedisCodePath    = "code/"
)

// Stores data about Token struct. Plaintext is return token value.
// Hash with token details for cache storage.
type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	Uuid      string    `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

// Creates new token.
func generateToken(uuid string, expiration time.Duration, key string, scope string) (*Token, error) {
	token := &Token{
		Uuid:   uuid,
		Expiry: time.Now().Add(expiration),
		Scope:  scope,
	}

	// TODO make better token generation
	randomBytes := make([]byte, 16)
	// fills the byte slice with random bytes from CSPRNG.
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	//encodeValue := append(randomBytes, token.Uuid...)
	//encodeValue = append(encodeValue, token.Expiry.String()...)
	encodeValue := append(randomBytes, key...)

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(encodeValue)[:32]

	hash := sha512.Sum512([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

// Returns authorization tokens.
func GetAuthTokens(uuid string, accessKey string, refreshKey string) (accessToken *Token, refreshToken *Token, err error) {

	accessToken, err = generateToken(uuid, 24*time.Hour, accessKey, ScopeAccess)
	if err != nil {
		return nil, nil, err
	}

	refreshToken, err = generateToken(uuid, 7*24*time.Hour, refreshKey, ScopeRefresh)
	if err != nil {
		return nil, nil, err
	}

	return
}

// Get User data by token.
func GetUserByToken(ctx context.Context, cache models.CacheClient, tokenPlainText string, user *models.User) error {
	//TODO token hash
	//tokenHash := sha512.Sum512([]byte(tokenPlainText))

	// string() type casting will return incorrect symbols.
	// hex.EncodeToString - should be replaced.
	//key := hex.EncodeToString(tokenHash[:])

	err := cache.GetHash(ctx, RedisAccessPath+tokenPlainText, user)
	if err != nil {
		// log: No such token.
		return errors.New("no such token")
	}

	return nil
}
