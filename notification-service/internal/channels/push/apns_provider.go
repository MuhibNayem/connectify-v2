package push

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/net/http2"
)

// APNSProvider implements PushProvider using Apple Push Notification Service HTTP/2 API
type APNSProvider struct {
	teamID     string
	keyID      string
	bundleID   string
	privateKey *ecdsa.PrivateKey
	production bool
	client     *http.Client
}

func NewAPNSProvider(teamID, keyID, bundleID, p8File string, production bool) (*APNSProvider, error) {
	// Parse the p8 file (PEM format)
	// Usually p8File is the content, or path? The arg name says File but struct said Content.
	// Assuming the argument passed is the CONTENT (string) based on usage context in cloud envs.
	// We'll trust caller passes content.
	block, _ := pem.Decode([]byte(p8File))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA private key")
	}

	// Setup HTTP/2 client
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}

	return &APNSProvider{
		teamID:     teamID,
		keyID:      keyID,
		bundleID:   bundleID,
		privateKey: ecdsaKey,
		production: production,
		client:     client,
	}, nil
}

func (a *APNSProvider) generateToken() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": a.teamID,
		"iat": time.Now().Unix(),
	})
	token.Header["kid"] = a.keyID

	return token.SignedString(a.privateKey)
}

func (a *APNSProvider) SendPush(ctx context.Context, token string, title, body string, data map[string]interface{}) error {
	host := "api.sandbox.push.apple.com"
	if a.production {
		host = "api.push.apple.com"
	}

	url := fmt.Sprintf("https://%s/3/device/%s", host, token)

	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"alert": map[string]string{
				"title": title,
				"body":  body,
			},
		},
		"custom_data": data,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	// Generate JWT token for APNS
	jwtToken, err := a.generateToken()
	if err != nil {
		return fmt.Errorf("failed to generate JWT: %w", err)
	}

	req.Header.Set("authorization", "bearer "+jwtToken)
	req.Header.Set("apns-topic", a.bundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("APNS API error: %d", resp.StatusCode)
	}

	return nil
}

func (a *APNSProvider) Name() string {
	return "apns"
}
