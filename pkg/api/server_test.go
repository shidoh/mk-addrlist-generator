package api

import (
	"encoding/json"
	"mk-addrlist-generator/pkg/config"
	"mk-addrlist-generator/pkg/generator"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestServer_HandleGetAllListsWithFormat(t *testing.T) {
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
		name           string
		format         string
		wantContains   string
		wantNotContain string
	}{
		{
			name:           "default mikrotik format",
			format:         "",
			wantContains:   "/ip/firewall/address-list",
			wantNotContain: "",
		},
		{
			name:           "mikrotik format explicit",
			format:         "mikrotik",
			wantContains:   "/ip/firewall/address-list",
			wantNotContain: "",
		},
		{
			name:           "plain format",
			format:         "plain",
			wantContains:   "192.168.1.1",
			wantNotContain: "/ip/firewall/address-list",
		},
		{
			name:           "json format",
			format:         "json",
			wantContains:   `"list_name"`,
			wantNotContain: "/ip/firewall/address-list",
		},
		{
			name:           "nftables format",
			format:         "nftables",
			wantContains:   "define test_v4",
			wantNotContain: "/ip/firewall/address-list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			url := "/lists/all"
			if tt.format != "" {
				url += "?format=" + tt.format
			}
			req, _ := http.NewRequest("GET", url, nil)
			server.router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("HandleGetAllLists() status = %v, want %v", w.Code, http.StatusOK)
			}

			body := w.Body.String()
			if tt.wantContains != "" && !strings.Contains(body, tt.wantContains) {
				t.Errorf("HandleGetAllLists() body should contain %q, got %q", tt.wantContains, body)
			}
			if tt.wantNotContain != "" && strings.Contains(body, tt.wantNotContain) {
				t.Errorf("HandleGetAllLists() body should not contain %q, got %q", tt.wantNotContain, body)
			}
		})
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

func TestServer_HandleGetListByNameWithFormat(t *testing.T) {
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
		name           string
		format         string
		wantContains   string
		wantNotContain string
	}{
		{
			name:           "default mikrotik format",
			format:         "",
			wantContains:   "/ip/firewall/address-list",
			wantNotContain: "",
		},
		{
			name:           "plain format",
			format:         "plain",
			wantContains:   "192.168.1.1",
			wantNotContain: "/ip/firewall/address-list",
		},
		{
			name:           "json format",
			format:         "json",
			wantContains:   `"address"`,
			wantNotContain: "/ip/firewall/address-list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			url := "/list/test"
			if tt.format != "" {
				url += "?format=" + tt.format
			}
			req, _ := http.NewRequest("GET", url, nil)
			server.router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("HandleGetListByName() status = %v, want %v", w.Code, http.StatusOK)
			}

			body := w.Body.String()
			if tt.wantContains != "" && !strings.Contains(body, tt.wantContains) {
				t.Errorf("HandleGetListByName() body should contain %q, got %q", tt.wantContains, body)
			}
			if tt.wantNotContain != "" && strings.Contains(body, tt.wantNotContain) {
				t.Errorf("HandleGetListByName() body should not contain %q, got %q", tt.wantNotContain, body)
			}
		})
	}
}

func TestGenerator_ParseFormat(t *testing.T) {
	tests := []struct {
		name       string
		formatStr  string
		wantFormat generator.OutputFormat
	}{
		{
			name:       "empty string defaults to mikrotik",
			formatStr:  "",
			wantFormat: generator.FormatMikrotik,
		},
		{
			name:       "mikrotik format",
			formatStr:  "mikrotik",
			wantFormat: generator.FormatMikrotik,
		},
		{
			name:       "plain format",
			formatStr:  "plain",
			wantFormat: generator.FormatPlain,
		},
		{
			name:       "json format",
			formatStr:  "json",
			wantFormat: generator.FormatJSON,
		},
		{
			name:       "nftables format",
			formatStr:  "nftables",
			wantFormat: generator.FormatNftables,
		},
		{
			name:       "unknown format defaults to mikrotik",
			formatStr:  "unknown",
			wantFormat: generator.FormatMikrotik,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generator.ParseFormat(tt.formatStr)
			if got != tt.wantFormat {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.formatStr, got, tt.wantFormat)
			}
		})
	}
}

func TestServer_HandleHealth(t *testing.T) {
	cfg := &config.Config{
		Config: config.ConfigDefaults{
			Timeout:       "1d",
			CommentPrefix: "test",
		},
		Lists: map[string]config.List{
			"test": {
				Addresses: []string{"192.168.1.1"},
			},
		},
	}

	server := NewServer(cfg)

	endpoints := []string{"/health", "/healthz"}
	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", endpoint, nil)
			server.router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("HandleHealth() status = %v, want %v", w.Code, http.StatusOK)
			}

			var response HealthResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to unmarshal health response: %v", err)
			}

			if response.Status != "healthy" {
				t.Errorf("HandleHealth() status = %v, want %v", response.Status, "healthy")
			}
		})
	}
}

func TestServer_HandleReady(t *testing.T) {
	cfg := &config.Config{
		Config: config.ConfigDefaults{
			Timeout:       "1d",
			CommentPrefix: "test",
		},
		Lists: map[string]config.List{
			"test": {
				Addresses: []string{"192.168.1.1"},
			},
		},
	}

	server := NewServer(cfg)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ready", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleReady() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestServer_HandleLists(t *testing.T) {
	cfg := &config.Config{
		Config: config.ConfigDefaults{
			Timeout:       "1d",
			CommentPrefix: "test",
		},
		Lists: map[string]config.List{
			"list1": {Addresses: []string{"192.168.1.1"}},
			"list2": {Addresses: []string{"10.0.0.1"}},
		},
	}

	server := NewServer(cfg)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/lists", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleListNames() status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "list1") || !strings.Contains(body, "list2") {
		t.Errorf("HandleListNames() should contain list names, got %q", body)
	}
}

func TestServer_HandleStats(t *testing.T) {
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
	req, _ := http.NewRequest("GET", "/stats", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleStats() status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "test") {
		t.Errorf("HandleStats() should contain list name 'test', got %q", body)
	}
}

func TestServer_HandleInfo(t *testing.T) {
	cfg := &config.Config{
		Config: config.ConfigDefaults{
			Timeout:       "1d",
			CommentPrefix: "test",
		},
		Lists: map[string]config.List{
			"test": {Addresses: []string{"192.168.1.1"}},
		},
	}

	server := NewServer(cfg)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/info", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleInfo() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response InfoResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal info response: %v", err)
	}

	if response.Name != "MikroTik Address List Generator" {
		t.Errorf("HandleInfo() name = %v, want %v", response.Name, "MikroTik Address List Generator")
	}

	if len(response.Formats) == 0 {
		t.Error("HandleInfo() formats should not be empty")
	}
}

func TestServer_Aggregate(t *testing.T) {
	cfg := &config.Config{
		Config: config.ConfigDefaults{
			Timeout:       "1d",
			CommentPrefix: "test",
		},
		Lists: map[string]config.List{
			"test": {
				Addresses: []string{
					"192.168.0.0/24",
					"192.168.1.0/24",
				},
			},
		},
	}

	server := NewServer(cfg)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/list/test?format=plain&aggregate=true", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Aggregate test status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	// With aggregation, these two /24s should become one /23
	if !strings.Contains(body, "192.168.0.0/23") {
		t.Errorf("Aggregate test should contain aggregated network, got %q", body)
	}
}
