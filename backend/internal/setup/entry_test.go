package setup

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCLIAdminPromptDefersDefaultsToBootstrap(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  AdminConfig
	}{
		{"both missing", "\n\n", AdminConfig{}},
		{"email missing", "\n12345678\n12345678\n", AdminConfig{Password: "12345678"}},
		{"password missing", "owner@example.com\n\n", AdminConfig{Email: "owner@example.com"}},
		{"whitespace defaults", "  \n  \n", AdminConfig{}},
		{"Unicode NEL defaults", "owner@example.com\n" + strings.Repeat("\u0085", 4) + "\n", AdminConfig{Email: "owner@example.com"}},
		{"Unicode BOM explicit", "owner@example.com\n" + strings.Repeat("\ufeff", 3) + "\n" + strings.Repeat("\ufeff", 3) + "\n", AdminConfig{Email: "owner@example.com", Password: strings.Repeat("\ufeff", 3)}},
		{"explicit password spaces", "owner@example.com\n 123456 \n 123456 \n", AdminConfig{Email: "owner@example.com", Password: " 123456 "}},
		{"validation retries", "invalid\nowner@example.com\nshort\n" + strings.Repeat("界", 25) + "\n12345678\nmismatch\n12345678\n12345678\n", AdminConfig{Email: "owner@example.com", Password: "12345678"}},
		{"UTF8 minimum bytes", "owner@example.com\n界界界\n界界界\n", AdminConfig{Email: "owner@example.com", Password: "界界界"}},
		{"bcrypt boundary", "owner@example.com\n" + strings.Repeat("界", 24) + "\n" + strings.Repeat("界", 24) + "\n", AdminConfig{Email: "owner@example.com", Password: strings.Repeat("界", 24)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tc.input + "next-answer\n"))
			admin := promptAdminConfig(reader)
			if admin != tc.want {
				t.Fatalf("admin=%#v, want %#v", admin, tc.want)
			}
			if next, err := reader.ReadString('\n'); err != nil || next != "next-answer\n" {
				t.Fatalf("credential prompts consumed a later answer: %q, %v", next, err)
			}
			emailGenerated, passwordGenerated, err := prepareAdminCredentials(&admin)
			if err != nil || emailGenerated != (tc.want.Email == "") || passwordGenerated != (tc.want.Password == "") {
				t.Fatalf("prepare flags=(%v,%v), error=%v", emailGenerated, passwordGenerated, err)
			}
			if tc.want.Email != "" && admin.Email != tc.want.Email || tc.want.Password != "" && admin.Password != tc.want.Password {
				t.Fatalf("shared preparation changed explicit credentials: %#v, want %#v", admin, tc.want)
			}
		})
	}
}

func TestInstallAdminDefaultsReachInstallation(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("SKIP_SETUP", "false")

	for _, tc := range []struct {
		name   string
		admin  AdminConfig
		status int
	}{
		{"both missing", AdminConfig{}, http.StatusInternalServerError},
		{"email missing", AdminConfig{Password: "12345678"}, http.StatusInternalServerError},
		{"password missing", AdminConfig{Email: "owner@example.com"}, http.StatusInternalServerError},
		{"whitespace defaults", AdminConfig{Email: "  ", Password: "  "}, http.StatusInternalServerError},
		{"explicit valid", AdminConfig{Email: "owner@example.com", Password: "12345678"}, http.StatusInternalServerError},
		{"bcrypt boundary", AdminConfig{Email: "owner@example.com", Password: strings.Repeat("界", 24)}, http.StatusInternalServerError},
		{"invalid email", AdminConfig{Email: "not-an-email"}, http.StatusBadRequest},
		{"unloginable email", AdminConfig{Email: "Owner <owner@example.com>"}, http.StatusBadRequest},
		{"short password", AdminConfig{Password: "1234567"}, http.StatusBadRequest},
		{"too many password bytes", AdminConfig{Password: strings.Repeat("界", 25)}, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// An owned local socket rejects PostgreSQL startup. A 500 proves the
			// real HTTP entry reached Install without requiring a database service.
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan struct{})
			go func() {
				defer close(done)
				conn, err := listener.Accept()
				if err == nil {
					_ = conn.Close()
				}
			}()
			t.Cleanup(func() {
				_ = listener.Close()
				<-done
			})

			addr, ok := listener.Addr().(*net.TCPAddr)
			if !ok {
				t.Fatalf("unexpected listener address type: %T", listener.Addr())
			}
			payload, err := json.Marshal(InstallRequest{
				Database: DatabaseConfig{Host: "127.0.0.1", Port: addr.Port, User: "test", DBName: "test", SSLMode: "disable"},
				Redis:    RedisConfig{Host: "127.0.0.1", Port: 6379},
				Admin:    tc.admin,
			})
			if err != nil {
				t.Fatal(err)
			}
			router := gin.New()
			RegisterRoutes(router)
			request := httptest.NewRequest(http.MethodPost, "/setup/install", bytes.NewReader(payload))
			request.Header.Set("Content-Type", "application/json")
			writer := httptest.NewRecorder()
			router.ServeHTTP(writer, request)
			if writer.Code != tc.status {
				t.Fatalf("status=%d, want %d; body=%s", writer.Code, tc.status, writer.Body.String())
			}
			if tc.status == http.StatusInternalServerError && !strings.Contains(writer.Body.String(), "database connection failed") {
				t.Fatalf("defaults should reach installation: %s", writer.Body.String())
			}
		})
	}
}
