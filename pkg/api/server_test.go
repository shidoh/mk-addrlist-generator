package api

import (
	"mk-addrlist-generator/pkg/config"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServer_HandleGetAllLists(t *testing.T) {
	cfg := &config.Config{
		Config: config.ConfigDefaults{
			Timeout:       "1d",
			CommentPrefix: "test",
		},
		Lists: map[string]config.List{
			"test": {
				Addresses: []string{
					"192.168.1.1",
					"10.0.0.0/24",
				},
			},
		},
	}

	server := NewServer(cfg)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/lists/all", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleGetAllLists() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestServer_HandleGetListByName(t *testing.T) {
	cfg := &config.Config{
		Config: config.ConfigDefaults{
			Timeout:       "1d",
			CommentPrefix: "test",
		},
		Lists: map[string]config.List{
			"test": {
				Addresses: []string{
					"192.168.1.1",
					"10.0.0.0/24",
				},
			},
		},
	}

	server := NewServer(cfg)

	tests := []struct {
		name       string
		listName   string
		wantStatus int
	}{
		{
			name:       "existing list",
			listName:   "test",
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent list",
			listName:   "nonexistent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/list/"+tt.listName, nil)
			server.router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleGetListByName() status = %v, want %v", w.Code, tt.wantStatus)
			}
		})
	}
}
