package pkg

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type TestAction struct {
	a string `redis:"a"`
	b int    `redis:"b"`
}

func client() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
	})
}

func TestActionManager_ConfirmNonExistent(t *testing.T) {

	db := client()

	manager := NewActionManager[TestAction](db, 32, "action:", time.Second*5)

	a, err := manager.ConfirmAction(context.Background(), "test")
	assert.NotNil(t, err)
	assert.Nil(t, a)
}

func put(t *testing.T, manager *ActionManager[TestAction]) string {
	action := &TestAction{
		a: "test",
		b: 1,
	}

	token, err := manager.RegisterAction(context.Background(), action)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	return token
}

func TestActionManager_FullFlow(t *testing.T) {
	db := client()

	manager := NewActionManager[TestAction](db, 32, "action:", time.Second*5)

	token := put(t, manager)

	a, err := manager.ConfirmAction(context.Background(), token)

	assert.NoError(t, err)
	assert.Equal(t, "test", a.a)
	assert.Equal(t, 1, a.b)
}

func TestActionManager_CancelAction(t *testing.T) {
	db := client()

	manager := NewActionManager[TestAction](db, 32, "action:", time.Second*5)
	token := put(t, manager)

	assert.NoError(t, manager.CancelAction(context.Background(), token))

	res, err := db.Exists(context.Background(), manager.RedisPrefix+token).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), res)
}

func TestActionManager_CancelNonExistent(t *testing.T) {
	db := client()

	manager := NewActionManager[TestAction](db, 32, "action:", time.Second*5)
	assert.NoError(t, manager.CancelAction(context.Background(), "test"))
}

func TestActionManager_TTL(t *testing.T) {
	db := client()

	manager := NewActionManager[TestAction](db, 32, "action:", time.Second*5)
	token := put(t, manager)

	time.Sleep(time.Second * 6)

	res, err := db.Exists(context.Background(), manager.RedisPrefix+token).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), res)
}
