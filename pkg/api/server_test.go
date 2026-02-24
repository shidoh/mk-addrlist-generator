package api

import (
	"mk-addrlist-generator/pkg/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestServer() (*gin.Engine, *Server) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	cfg := &config.Config{
		Config: config.GlobalConfig{
			Timeout:       "1h",
			CommentPrefix: "Global comment",
		},
		Lists: map[string]config.List{
			"test1": {
				Addresses: []string{"192.168.1.0/24"},
			},
			"test2": {
				Addresses: []string{"10.0.0.0/8"},
			},
		},
	}

	server := NewServer(cfg)
	server.SetupRoutes(r)

	return r, server
}

func TestServer_HandleGetAllLists(t *testing.T) {
	r, _ := setupTestServer()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/lists/all", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	expectedParts := []string{
		"test1",
		"test2",
		"192.168.1.0/24",
		"10.0.0.0/8",
	}

	for _, part := range expectedParts {
		if !strings.Contains(body, part) {
			t.Errorf("Response missing expected part %q", part)
		}
	}
}

func TestServer_HandleGetListByName(t *testing.T) {
	r, _ := setupTestServer()

	tests := []struct {
		name       string
		listName   string
		wantStatus int
		wantParts  []string
	}{
		{
			name:       "existing list",
			listName:   "test1",
			wantStatus: http.StatusOK,
			wantParts:  []string{"test1", "192.168.1.0/24"},
		},
		{
			name:       "non-existent list",
			listName:   "nonexistent",
			wantStatus: http.StatusNotFound,
			wantParts:  []string{"not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/list/"+tt.listName, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status code %d, got %d", tt.wantStatus, w.Code)
			}

			body := w.Body.String()
			for _, part := range tt.wantParts {
				if !strings.Contains(body, part) {
					t.Errorf("Response missing expected part %q", part)
				}
			}
		})
	}
}