package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/securestor/securestor/internal/models"
	"golang.org/x/oauth2"
)

// OIDCService handles OpenID Connect authentication
type OIDCService struct {
	provider     *oidc.Provider
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
	config       *models.OIDCConfig
	logger       *log.Logger
}

// NewOIDCService creates a new OIDC service
func NewOIDCService(config *models.OIDCConfig, logger *log.Logger) (*OIDCService, error) {
	ctx := context.Background()

	// Initialize OIDC provider
	provider, err := oidc.NewProvider(ctx, config.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	// Configure OAuth2
	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       config.Scopes,
	}

	// Configure ID token verifier
	verifier := provider.Verifier(&oidc.Config{
		ClientID: config.ClientID,
	})

	return &OIDCService{
		provider:     provider,
		oauth2Config: oauth2Config,
		verifier:     verifier,
		config:       config,
		logger:       logger,
	}, nil
}

// GenerateState generates a random state string for OIDC flows
func (s *OIDCService) GenerateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetAuthCodeURL returns the authorization URL for the OIDC flow
func (s *OIDCService) GetAuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return s.oauth2Config.AuthCodeURL(state, opts...)
}

// ExchangeToken exchanges an authorization code for tokens
func (s *OIDCService) ExchangeToken(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}
	return token, nil
}

// VerifyIDToken verifies and parses an ID token
func (s *OIDCService) VerifyIDToken(ctx context.Context, rawIDToken string) (*models.TokenClaims, error) {
	idToken, err := s.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	var claims models.TokenClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse ID token claims: %w", err)
	}

	return &claims, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *OIDCService) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := s.oauth2Config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return newToken, nil
}

// GetUserInfo retrieves user information from the OIDC provider
func (s *OIDCService) GetUserInfo(ctx context.Context, accessToken string) (*oidc.UserInfo, error) {
	userInfo, err := s.provider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return userInfo, nil
}

// GetLogoutURL returns the logout URL for the OIDC provider
func (s *OIDCService) GetLogoutURL(idTokenHint string) string {
	logoutURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/logout",
		s.config.IssuerURL, "securestor")

	if idTokenHint != "" {
		logoutURL += "?id_token_hint=" + idTokenHint
		if s.config.PostLogoutURL != "" {
			logoutURL += "&post_logout_redirect_uri=" + s.config.PostLogoutURL
		}
	}

	return logoutURL
}

// ValidateAccessToken validates an access token
func (s *OIDCService) ValidateAccessToken(ctx context.Context, accessToken string) (*models.TokenClaims, error) {
	// For Keycloak, we can validate the access token by getting user info
	userInfo, err := s.GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	var claims models.TokenClaims
	if err := userInfo.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse access token claims: %w", err)
	}

	return &claims, nil
}

// CreateOIDCConfig creates an OIDC configuration from environment variables
func CreateOIDCConfig() *models.OIDCConfig {
	return &models.OIDCConfig{
		IssuerURL:     "http://localhost:8090/realms/securestor",
		ClientID:      "securestor-api",
		ClientSecret:  "securestor-api-secret-key-2024",
		RedirectURL:   "http://localhost:8080/api/v1/auth/callback",
		PostLogoutURL: "http://localhost:3000",
		Scopes:        []string{oidc.ScopeOpenID, "profile", "email", "roles"},
	}
}
