//go:build modeltrace_postgres && !integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	_ "github.com/lib/pq"
)

var integrationDB *sql.DB
var integrationEntClient *dbent.Client

func testEntClient(t *testing.T) *dbent.Client {
	t.Helper()
	return integrationEntClient
}

func TestMain(m *testing.M) { os.Exit(runModelTracePostgres(m)) }

func runModelTracePostgres(m *testing.M) (result int) {
	if os.Getenv("SUB2API_MODELTRACE_POSTGRES") != "1" {
		log.Print("set SUB2API_MODELTRACE_POSTGRES=1 to enable local PostgreSQL test")
		return 1
	}
	ctx := context.Background()
	if err := timezone.Init("UTC"); err != nil {
		log.Printf("initialize timezone: %v", err)
		return 1
	}
	admin, err := sql.Open("postgres", "host=127.0.0.1 port=55479 user=postgres dbname=postgres sslmode=disable")
	if err != nil {
		log.Printf("open admin database: %v", err)
		return 1
	}
	defer admin.Close()
	if err := admin.PingContext(ctx); err != nil {
		log.Printf("ping admin database: %v", err)
		return 1
	}
	name := fmt.Sprintf("sub2api_modeltrace_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		log.Printf("create %s: %v", name, err)
		return 1
	}
	log.Printf("created isolated database %s", name)
	defer func() {
		if _, err := admin.ExecContext(ctx, "DROP DATABASE "+name); err != nil {
			log.Printf("drop %s: %v", name, err)
			result = 1
		} else {
			log.Printf("dropped isolated database %s", name)
		}
	}()
	integrationDB, err = sql.Open("postgres", fmt.Sprintf("host=127.0.0.1 port=55479 user=postgres dbname=%s sslmode=disable TimeZone=UTC", name))
	if err != nil {
		log.Printf("open test database: %v", err)
		return 1
	}
	defer integrationDB.Close()
	if err := ApplyMigrations(ctx, integrationDB); err != nil {
		log.Printf("apply migrations: %v", err)
		return 1
	}
	integrationEntClient = dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, integrationDB)))
	return m.Run()
}
