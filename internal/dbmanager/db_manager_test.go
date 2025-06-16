package dbmanager

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	code, err := runMain(m)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(code)
}

const testDefaultTimeout = 3 * time.Second

const (
	testDBName       = "test"
	testUserName     = "test"
	testUserPassword = "test"
)

var (
	getSUConnection func() (*pgx.Conn, error)
	getDSN          func() string
)

func initGetDSN(hostPort string) {
	getDSN = func() string {
		return fmt.Sprintf(
			"postgres://%s:%s@%s/%s?sslmode=disable",
			testUserName,
			testUserPassword,
			hostPort,
			testDBName,
		)
	}
}

func initGetSUConnection(hostPort string) {
	getSUConnection = func() (*pgx.Conn, error) {
		dsn := fmt.Sprintf(
			"postgres://%s:%s@%s/%s?sslmode=disable",
			"postgres",
			"postgres",
			hostPort,
			"postgres",
		)
		conn, err := pgx.Connect(context.TODO(), dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to get a super user connection: %w", err)
		}

		return conn, nil
	}
}

func loadImageFromEnv() string {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("error loading .env file: %v", err)
	}
	return os.Getenv("POSTGRES_TAG")
}

func runMain(m *testing.M) (int, error) {
	dockerPool, err := dockertest.NewPool("")
	if err != nil {
		return 1, fmt.Errorf("failed to initialize a docker pool: %w", err)
	}

	const pgPort = "5432/tcp"
	pgContainer, err := dockerPool.RunWithOptions(
		&dockertest.RunOptions{
			Name:       "migrations-integration-tests",
			Repository: "postgres",
			Tag:        loadImageFromEnv(),
			Env: []string{
				"POSTGRES_USER=postgres",
				"POSTGRES_PASSWORD=postgres",
			},
			ExposedPorts: []string{pgPort},
		},
		func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		},
	)
	if err != nil {
		return 1, fmt.Errorf("failed to run postgres container: %w", err)
	}
	defer func() {
		if err := dockerPool.Purge(pgContainer); err != nil {
			log.Printf("failed to purge the postgres container: %v", err)
		}
	}()

	hostPort := pgContainer.GetHostPort(pgPort)
	initGetDSN(hostPort)
	initGetSUConnection(hostPort)

	dockerPool.MaxWait = 10 * time.Second
	var conn *pgx.Conn
	if err := dockerPool.Retry(func() error {
		conn, err = getSUConnection()
		if err != nil {
			return fmt.Errorf("failed to connect to the DB: %w", err)
		}
		return nil
	}); err != nil {
		return 1, fmt.Errorf("retry failed: %w", err)
	}
	defer func() {
		if err := conn.Close(context.TODO()); err != nil {
			log.Printf("failed to correctly close the DB connection: %v", err)
		}
	}()

	if err := createTestDB(conn); err != nil {
		return 1, fmt.Errorf("failed to create a test DB: %w", err)
	}

	exitCode := m.Run()

	return exitCode, nil
}

func createTestDB(conn *pgx.Conn) error {
	const (
		createUser = `CREATE USER %s PASSWORD '%s';`
		createDB   = `CREATE DATABASE %s
		OWNER %s
		ENCODING 'UTF8'
		LC_COLLATE = 'en_US.utf8'
		LC_CTYPE = 'en_US.utf8';`
	)

	ctx, cancel1 := context.WithTimeout(context.Background(), testDefaultTimeout)
	defer cancel1()
	_, err := conn.Exec(ctx, fmt.Sprintf(createUser, testUserName, testUserPassword))
	if err != nil {
		return fmt.Errorf("failed to create a test user: %w", err)
	}

	ctx, cancel2 := context.WithTimeout(context.Background(), testDefaultTimeout)
	defer cancel2()
	_, err = conn.Exec(ctx, fmt.Sprintf(createDB, testDBName, testUserName))
	if err != nil {
		return fmt.Errorf("failed to create a test DB: %w", err)
	}

	return nil
}

func TestDBManager_Connect(t *testing.T) {
	dsn := getDSN()
	db := New(dsn, slog.Default())
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testDefaultTimeout)
	defer cancel()

	db.Connect(ctx)
	if err := db.Error(); err != nil {
		t.Errorf("failed to connect to test DB using dsn %s: %v", dsn, err)
	}
}

func TestDBManager_Ping(t *testing.T) {
	dsn := getDSN()
	db := New(dsn, slog.Default())
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testDefaultTimeout)
	defer cancel()

	db.Connect(ctx).Ping(ctx)
	if err := db.Error(); err != nil {
		t.Errorf("failed to ping test DB using dsn %s: %v", dsn, err)
	}
}

func TestDBManager_ApplyMigrations(t *testing.T) {
	dsn := getDSN()
	db := New(dsn, slog.Default())
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testDefaultTimeout)
	defer cancel()

	db.Connect(ctx).Ping(ctx).ApplyMigrations(ctx).ApplyMigrations(ctx)
	if err := db.Error(); err != nil {
		t.Errorf("failed to apply mirgrations to test db using dsn %s: %v", dsn, err)
	}
}

func TestDBManager_GetPool_from_nil(t *testing.T) {
	dsn := getDSN()
	db := New(dsn, slog.Default())
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testDefaultTimeout)
	defer cancel()
	p, err := db.GetPool(ctx)
	assert.Nil(t, p)
	assert.Error(t, err)
}

func TestDBManager_GetPool(t *testing.T) {
	dsn := getDSN()
	db := New(dsn, slog.Default())
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testDefaultTimeout)
	defer cancel()

	db.Connect(ctx)
	if err := db.Error(); err != nil {
		t.Errorf("failed to connect to test DB using dsn %s: %v", dsn, err)
	}
	p, err := db.GetPool(ctx)
	require.NoError(t, err)
	assert.NotNil(t, p)
}
