package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostgresConfig_DSN(t *testing.T) {
	config := PostgresConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "admin",
		Password: "secret",
		Name:     "testdb",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=admin password=secret dbname=testdb sslmode=disable TimeZone=UTC"
	assert.Equal(t, expected, config.DSN())
}

func TestPostgresConfig_DSN_WithDifferentValues(t *testing.T) {
	config := PostgresConfig{
		Host:     "db.example.com",
		Port:     "5433",
		User:     "appuser",
		Password: "p@ssw0rd!",
		Name:     "production",
		SSLMode:  "require",
	}

	expected := "host=db.example.com port=5433 user=appuser password=p@ssw0rd! dbname=production sslmode=require TimeZone=UTC"
	assert.Equal(t, expected, config.DSN())
}

func TestPostgresConfig_DSN_WithEmptyFields(t *testing.T) {
	config := PostgresConfig{}

	expected := "host= port= user= password= dbname= sslmode= TimeZone=UTC"
	assert.Equal(t, expected, config.DSN())
}

func TestPostgresConfig_DSN_ContainsTimeZone(t *testing.T) {
	config := PostgresConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "user",
		Password: "pass",
		Name:     "db",
		SSLMode:  "disable",
	}

	dsn := config.DSN()
	assert.Contains(t, dsn, "TimeZone=UTC")
}

func TestPostgresConfig_DSN_ContainsAllFields(t *testing.T) {
	config := PostgresConfig{
		Host:     "myhost",
		Port:     "5432",
		User:     "myuser",
		Password: "mypass",
		Name:     "mydb",
		SSLMode:  "verify-full",
	}

	dsn := config.DSN()
	assert.Contains(t, dsn, "host=myhost")
	assert.Contains(t, dsn, "port=5432")
	assert.Contains(t, dsn, "user=myuser")
	assert.Contains(t, dsn, "password=mypass")
	assert.Contains(t, dsn, "dbname=mydb")
	assert.Contains(t, dsn, "sslmode=verify-full")
}

func TestErrPostgresConnectionFailed(t *testing.T) {
	assert.NotNil(t, ErrPostgresConnectionFailed)
	assert.Equal(t, "database connection failed", ErrPostgresConnectionFailed.Error())
}

func TestErrPostgresPingFailed(t *testing.T) {
	assert.NotNil(t, ErrPostgresPingFailed)
	assert.Equal(t, "database ping failed", ErrPostgresPingFailed.Error())
}

func TestPostgresErrors_AreDistinct(t *testing.T) {
	assert.NotEqual(t, ErrPostgresConnectionFailed, ErrPostgresPingFailed)
}
