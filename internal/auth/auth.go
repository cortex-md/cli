package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	KeyringService = "cortex-cli"
	KeyringUser    = "github-token"
)

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

var (
	ErrAuthPending     = errors.New("authorization pending")
	ErrAuthExpired     = errors.New("device code expired")
	ErrAuthDenied      = errors.New("authorization denied")
	ErrSlowDown        = errors.New("slow down")
	ErrNoToken         = errors.New("no token found")
	ErrInvalidClientID = errors.New("invalid client id")
)

type Client struct {
	clientID   string
	httpClient *http.Client
}

func NewClient(clientID string) *Client {
	return &Client{
		clientID: clientID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) StartDeviceFlow(ctx context.Context) (*DeviceCodeResponse, error) {
	if c.clientID == "" {
		return nil, ErrInvalidClientID
	}

	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("scope", "repo read:user")

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://github.com/login/device/code",
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.URL.RawQuery = data.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var deviceResp DeviceCodeResponse
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		return nil, err
	}

	return &deviceResp, nil
}

func (c *Client) PollForToken(ctx context.Context, deviceCode string, interval int) (string, error) {
	currentInterval := interval
	if currentInterval < 5 {
		currentInterval = 5
	}

	ticker := time.NewTicker(time.Duration(currentInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			token, err := c.checkDeviceAuth(ctx, deviceCode)
			if err == nil {
				return token, nil
			}

			if err == ErrAuthPending {
				continue
			}

			if err == ErrSlowDown {
				ticker.Stop()
				currentInterval += 5
				ticker = time.NewTicker(time.Duration(currentInterval) * time.Second)
				continue
			}

			return "", err
		}
	}
}

func (c *Client) checkDeviceAuth(ctx context.Context, deviceCode string) (string, error) {
	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("device_code", deviceCode)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://github.com/login/oauth/access_token",
		nil,
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	req.URL.RawQuery = data.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp AccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	if tokenResp.Error != "" {
		switch tokenResp.Error {
		case "authorization_pending":
			return "", ErrAuthPending
		case "slow_down":
			return "", ErrSlowDown
		case "expired_token":
			return "", ErrAuthExpired
		case "access_denied":
			return "", ErrAuthDenied
		default:
			return "", fmt.Errorf("auth error: %s", tokenResp.Error)
		}
	}

	return tokenResp.AccessToken, nil
}

func SaveToken(token string) error {
	return keyring.Set(KeyringService, KeyringUser, token)
}

func GetToken() (string, error) {
	token, err := keyring.Get(KeyringService, KeyringUser)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", ErrNoToken
		}
		return "", err
	}
	return token, nil
}

func DeleteToken() error {
	err := keyring.Delete(KeyringService, KeyringUser)
	if err != nil && err != keyring.ErrNotFound {
		return err
	}
	return nil
}

func IsAuthenticated() bool {
	_, err := GetToken()
	return err == nil
}

func ResolveToken() (string, error) {
	envToken := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if envToken != "" {
		return envToken, nil
	}

	storedToken, err := GetToken()
	if err == nil {
		return storedToken, nil
	}

	if errors.Is(err, ErrNoToken) {
		return "", ErrNoToken
	}

	return "", err
}

func CopyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	case "windows":
		cmd = exec.Command("cmd", "/c", "clip")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	pipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	if _, err := pipe.Write([]byte(text)); err != nil {
		return err
	}

	if err := pipe.Close(); err != nil {
		return err
	}

	return cmd.Wait()
}

func OpenBrowser(urlStr string) error {
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", urlStr)
	case "linux":
		cmd = exec.Command("xdg-open", urlStr)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", urlStr)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
