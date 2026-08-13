package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/service"
)

func TestPublicKeyConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		config  config.PublicKeyAuthConfig
		enabled bool
	}{
		{name: "disabled", config: config.PublicKeyAuthConfig{}, enabled: false},
		{
			name: "enabled",
			config: config.PublicKeyAuthConfig{
				Enabled: true, RPID: "localhost", RPDisplayName: "KubeVision",
				Origins: []string{"http://localhost:5173"}, UserVerification: "preferred",
				CounterPolicy: "deny", ChallengeTTL: 5 * time.Minute,
			},
			enabled: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			publicKeyService, err := service.NewPublicKeyService(nil, nil, nil, &config.Config{Auth: config.AuthConfig{PublicKey: test.config}}, zap.NewNop())
			require.NoError(t, err)

			router := gin.New()
			router.GET("/api/v1/auth/public-key/config", NewPublicKeyHandler(publicKeyService).Config)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/auth/public-key/config", nil))

			require.Equal(t, http.StatusOK, response.Code)
			var body struct {
				Code int `json:"code"`
				Data struct {
					Enabled bool `json:"enabled"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			require.Zero(t, body.Code)
			require.Equal(t, test.enabled, body.Data.Enabled)
		})
	}
}
