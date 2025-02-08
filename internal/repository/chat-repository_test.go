package repository

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// Unit Tests
func setupTestRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Test DB
	})
}

func TestChatRepository_AddUser(t *testing.T) {
	client := setupTestRedisClient()
	repo := NewRedisClient(client)
	ctx := context.Background()

	err := repo.AddUser(ctx, 123)
	assert.NoError(t, err)

	users, err := repo.GetUsers(ctx)
	assert.NoError(t, err)
	assert.Contains(t, users, int64(123))

	client.FlushDB(ctx)
}

func TestChatRepository_FindPartner(t *testing.T) {
	client := setupTestRedisClient()
	repo := NewRedisClient(client)
	ctx := context.Background()

	repo.AddUser(ctx, 123)
	repo.AddUser(ctx, 456)

	partner, err := repo.FindPartner(ctx, 123)
	assert.NoError(t, err)
	assert.Equal(t, int64(456), partner)

	users, err := repo.GetUsers(ctx)
	assert.NoError(t, err)
	assert.NotContains(t, users, int64(456))

	client.FlushDB(ctx)
}

func TestChatRepository_SetAndGetPartner(t *testing.T) {
	client := setupTestRedisClient()
	repo := NewRedisClient(client)
	ctx := context.Background()

	repo.SetPartner(ctx, 123, 456)

	partner, err := repo.GetUserPartner(ctx, 123)
	assert.NoError(t, err)
	assert.Equal(t, int64(456), partner)

	client.FlushDB(ctx)
}

func TestChatRepository_RemoveUser(t *testing.T) {
	client := setupTestRedisClient()
	repo := NewRedisClient(client)
	ctx := context.Background()

	repo.AddUser(ctx, 123)
	repo.RemoveUser(ctx, 123)

	users, err := repo.GetUsers(ctx)
	assert.NoError(t, err)
	assert.NotContains(t, users, int64(123))

	partner, err := repo.GetUserPartner(ctx, 123)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), partner)

	client.FlushDB(ctx)
}

func TestChatRepository_GetUsers(t *testing.T) {
	client := setupTestRedisClient()
	repo := NewRedisClient(client)
	ctx := context.Background()

	repo.AddUser(ctx, 123)
	repo.AddUser(ctx, 456)

	users, err := repo.GetUsers(ctx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []int64{123, 456}, users)

	client.FlushDB(ctx)
}
