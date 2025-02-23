package pkg

import (
	"context"
	"crypto/rand"
	"github.com/redis/go-redis/v9"
	"time"
	"unsafe"
)

type ActionManager[A any] struct {
	TokenLength int
	RedisPrefix string
	TTL         time.Duration

	rdb *redis.Client
}

func NewActionManager[A any](rdb *redis.Client, tokenLength int, prefix string, ttl time.Duration) *ActionManager[A] {
	return &ActionManager[A]{
		TokenLength: tokenLength,
		RedisPrefix: prefix,
		TTL:         ttl,
		rdb:         rdb,
	}
}

func (m *ActionManager[A]) RegisterAction(ctx context.Context, action *A) (string, error) {
	token, err := m.newToken()

	if err != nil {
		return "", err
	}

	key := m.RedisPrefix + token

	tx := m.rdb.TxPipeline()
	res := tx.HSet(ctx, key, action)
	tx.Expire(ctx, key, m.TTL)
	_, err = tx.Exec(ctx)
	if err != nil {
		return "", err
	}

	if res.Err() != nil {
		return "", res.Err()
	}

	return token, nil
}

func (m *ActionManager[A]) CancelAction(ctx context.Context, token string) error {
	key := m.RedisPrefix + token
	_, err := m.rdb.Del(ctx, key).Result()
	return err
}

func (m *ActionManager[A]) ConfirmAction(ctx context.Context, token string) (*A, error) {
	key := m.RedisPrefix + token
	tx := m.rdb.TxPipeline()

	res := tx.HGetAll(ctx, key)
	tx.Del(ctx, key)

	_, err := tx.Exec(ctx)

	if err != nil {
		return nil, err
	}

	if res.Err() != nil {
		return nil, res.Err()
	}

	var action A

	err = res.Scan(&action)

	return &action, err
}

func (m *ActionManager[A]) newToken() (string, error) {
	b := make([]byte, m.TokenLength)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return *((*string)(unsafe.Pointer(&b))), nil
}
