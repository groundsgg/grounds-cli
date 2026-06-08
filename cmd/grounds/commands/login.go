package commands

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/browser"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/render"
)

const (
	defaultIssuer   = "https://account.grounds.gg/realms/grounds"
	defaultClientID = "grounds-cli"
)

func NewLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate via OAuth device flow",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load("")
			if err != nil {
				return err
			}
			device := &auth.DeviceClient{
				Issuer:   defaultIssuer,
				ClientID: defaultClientID,
				HTTP:     defaultHTTP(),
			}
			ctx := context.Background()
			dc, err := device.StartDevice(ctx)
			if err != nil {
				return err
			}

			printDeviceLoginInstructions(cmd.OutOrStdout(), dc, browser.OpenURL(dc.VerificationURIComplete))

			tok, err := device.PollToken(ctx, dc.DeviceCode, dc.CodeVerifier, dc.Interval, dc.ExpiresIn)
			if err != nil {
				return err
			}

			// ID-token claims (best-effort decode).
			email, preferred := decodeIDToken(tok.IDToken)
			creds := auth.CredentialsFromToken(tok, email, preferred)

			store := auth.NewStore(cfg.Dir)
			if err := store.Save(creds); err != nil {
				return err
			}

			render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Auth", "Logged in as "+loginSubject(preferred, email))
			render.DetailLine(cmd.OutOrStdout(), render.StatusOK,
				"New here? Run "+render.Command("grounds onboarding")+" for a guided setup.")
			return nil
		},
	}
}

func printDeviceLoginInstructions(out io.Writer, dc *auth.DeviceCodeResponse, openErr error) {
	if openErr != nil {
		render.StatusLine(out, render.StatusWarn, "Browser", "Could not open device login page automatically")
		render.DetailLine(out, render.StatusWarn, "URL: "+dc.VerificationURI)
		render.DetailLine(out, render.StatusWarn, "Code: "+dc.UserCode)
		return
	}
	render.StatusLine(out, render.StatusOK, "Browser", "Opened device login page")
	render.DetailLine(out, render.StatusOK, "Code: "+dc.UserCode)
}

func loginSubject(preferred, email string) string {
	if preferred != "" {
		return preferred
	}
	if email != "" {
		return email
	}
	return "current user"
}

func decodeIDToken(idToken string) (email, preferred string) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return "", ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ""
	}
	var claims struct {
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", ""
	}
	return claims.Email, claims.PreferredUsername
}

func defaultHTTP() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}
