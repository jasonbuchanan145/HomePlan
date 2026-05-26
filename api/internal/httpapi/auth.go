package httpapi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"homeplan/api/internal/identity"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/google"
)

const homeplanAppKey = "homeplan"

type AuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	BaseURL            string
	SessionCookieName  string
	SessionTTL         time.Duration
}

type AuthUser = identity.AuthUser
type AppEntitlement = identity.AppEntitlement
type AuthIdentity = identity.AuthIdentity
type SignedInSession = identity.SignedInSession

type AuthProvider interface {
	Name() string
	BeginAuth(state string) (AuthProviderSession, error)
	CompleteAuth(session string, params url.Values) (AuthIdentity, error)
}

type AuthProviderSession struct {
	AuthURL string
	Session string
}

type gothAuthProvider struct {
	provider goth.Provider
}

func newGoogleAuthProvider(config AuthConfig) AuthProvider {
	provider := google.New(
		config.GoogleClientID,
		config.GoogleClientSecret,
		googleRedirectURL(config.BaseURL),
		"openid",
		"email",
		"profile",
	)
	provider.SetPrompt("select_account")
	return gothAuthProvider{provider: provider}
}

func (provider gothAuthProvider) Name() string {
	return provider.provider.Name()
}

func (provider gothAuthProvider) BeginAuth(state string) (AuthProviderSession, error) {
	session, err := provider.provider.BeginAuth(state)
	if err != nil {
		return AuthProviderSession{}, err
	}
	authURL, err := session.GetAuthURL()
	if err != nil {
		return AuthProviderSession{}, err
	}
	return AuthProviderSession{AuthURL: authURL, Session: session.Marshal()}, nil
}

func (provider gothAuthProvider) CompleteAuth(sessionValue string, params url.Values) (AuthIdentity, error) {
	session, err := provider.provider.UnmarshalSession(sessionValue)
	if err != nil {
		return AuthIdentity{}, err
	}
	if _, err := session.Authorize(provider.provider, params); err != nil {
		return AuthIdentity{}, err
	}
	user, err := provider.provider.FetchUser(session)
	if err != nil {
		return AuthIdentity{}, err
	}
	if user.UserID == "" || user.Email == "" {
		return AuthIdentity{}, errors.New("provider profile missing subject or email")
	}
	emailVerified := true
	if value, ok := user.RawData["verified_email"].(bool); ok {
		emailVerified = value
	}
	return AuthIdentity{
		Provider:        user.Provider,
		ProviderSubject: user.UserID,
		Email:           strings.ToLower(user.Email),
		EmailVerified:   emailVerified,
		DisplayName:     user.Name,
		AvatarURL:       user.AvatarURL,
	}, nil
}

func encodeOAuthSessionCookie(session string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(session))
}

func decodeOAuthSessionCookie(value string) (string, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("decode oauth session cookie: %w", err)
	}
	return string(bytes), nil
}

func googleRedirectURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/api/auth/google/callback"
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
