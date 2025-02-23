package pkg

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var db = client()
var manager = NewActionManager[TestAction](db, 32, "action:", time.Second*5)

type TestAction struct {
	a string `redis:"a"`
	b int    `redis:"b"`
}

func client() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}

func TestActionManager_ConfirmNonExistent(t *testing.T) {
	a, err := manager.ConfirmAction(context.Background(), "test-nonexistent")
	assert.ErrorIs(t, err, NonExistentAction)
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

func TestActionManager_CancelAction(t *testing.T) {
	token := put(t, manager)

	assert.NoError(t, manager.CancelAction(context.Background(), token))

	res, err := db.Exists(context.Background(), manager.RedisPrefix+token).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), res)
}

func TestActionManager_CancelNonExistent(t *testing.T) {
	assert.NoError(t, manager.CancelAction(context.Background(), "test-nonexistent"))
}

func TestActionManager_TTL(t *testing.T) {
	token := put(t, manager)

	time.Sleep(time.Second * 6)

	res, err := db.Exists(context.Background(), manager.RedisPrefix+token).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), res)
}
