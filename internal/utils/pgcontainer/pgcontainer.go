package pgcontainer

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	"github.com/talx-hub/gopher-bonus/internal/model"
)

const (
	testDBName       = "test"
	testUserName     = "test"
	testUserPassword = "test"
)

const queryTimeOut = 5 * time.Second
const retryTimeout = 10 * time.Second

type PGContainer struct {
	dockerPool  *dockertest.Pool
	pgContainer *dockertest.Resource
	log         *slog.Logger
	dsn         string
}

func New(log *slog.Logger) *PGContainer {
	return &PGContainer{
		log: log,
	}
}

func (c *PGContainer) GetDSN() string {
	return c.dsn
}

func (c *PGContainer) Close() {
	if err := c.dockerPool.Purge(c.pgContainer); err != nil {
		c.log.ErrorContext(context.TODO(),
			"failed to purge the postgres container",
			slog.Any(model.KeyLoggerError, err))
	}
}

func (c *PGContainer) RunContainer() error {
	var err error
	c.dockerPool, err = dockertest.NewPool("")
	if err != nil {
		return fmt.Errorf("failed to initialize a docker pool: %w", err)
	}

	const pgPort = "5432/tcp"
	c.pgContainer, err = c.dockerPool.RunWithOptions(
		&dockertest.RunOptions{
			Name:       "migrations-integration-tests",
			Repository: "postgres",
			Tag:        c.loadImageFromEnv(),
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
		return fmt.Errorf("failed to run postgres container: %w", err)
	}

	hostPort := c.pgContainer.GetHostPort(pgPort)

	c.dockerPool.MaxWait = retryTimeout
	var conn *pgx.Conn
	if err := c.dockerPool.Retry(func() error {
		conn, err = getSUConnection(hostPort)
		if err != nil {
			return fmt.Errorf("failed to connect to the DB: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("retry failed: %w", err)
	}
	defer func() {
		if err = conn.Close(context.TODO()); err != nil {
			c.log.ErrorContext(context.TODO(),
				"failed to correctly close the DB connection",
				slog.Any(model.KeyLoggerError, err),
			)
		}
	}()

	if c.dsn, err = createTestDB(conn); err != nil {
		return fmt.Errorf("failed to create a test DB: %w", err)
	}
	return nil
}

func (c *PGContainer) loadImageFromEnv() string {
	if err := godotenv.Load(".env"); err != nil {
		c.log.ErrorContext(context.TODO(),
			"error loading .env file: %v",
			slog.Any(model.KeyLoggerError, err),
		)
	}
	return os.Getenv("POSTGRES_TAG")
}

func getSUConnection(hostPort string) (*pgx.Conn, error) {
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

func createTestDB(conn *pgx.Conn) (string, error) {
	const (
		createUser = `CREATE USER %s PASSWORD '%s';`
		createDB   = `CREATE DATABASE %s
		OWNER %s
		ENCODING 'UTF8'
		LC_COLLATE = 'en_US.utf8'
		LC_CTYPE = 'en_US.utf8';`
	)

	ctx, cancel1 := context.WithTimeout(context.Background(), queryTimeOut)
	defer cancel1()
	_, err := conn.Exec(ctx, fmt.Sprintf(createUser, testUserName, testUserPassword))
	if err != nil {
		return "", fmt.Errorf("failed to create a test user: %w", err)
	}

	ctx, cancel2 := context.WithTimeout(context.Background(), queryTimeOut)
	defer cancel2()
	_, err = conn.Exec(ctx, fmt.Sprintf(createDB, testDBName, testUserName))
	if err != nil {
		return "", fmt.Errorf("failed to create a test DB: %w", err)
	}

	testDatabaseDSN := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		testUserName,
		testUserPassword,
		net.JoinHostPort(
			conn.Config().Host,
			strconv.FormatUint(uint64(conn.Config().Port), 10)),
		testDBName,
	)

	return testDatabaseDSN, nil
}
