package pkg

import (
	"context"
	"crypto/rand"
	"errors"
	"github.com/redis/go-redis/v9"
	"time"
	"unsafe"
)

var NonExistentAction = errors.New("non existent action")

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

func (m *ActionManager[A]) transaction(ctx context.Context, fn func(redis.Pipeliner) error) error {
	cmds, err := m.rdb.TxPipelined(ctx, fn)
	if err != nil {
		return err
	}
	errs := make([]error, len(cmds)+1)

	for i, cmd := range cmds {
		errs[i] = cmd.Err()
	}

	errs[len(errs)-1] = err

	return errors.Join(errs...)
}

func (m *ActionManager[A]) RegisterAction(ctx context.Context, action *A) (string, error) {
	token, err := m.newToken()

	if err != nil {
		return "", err
	}

	key := m.RedisPrefix + token

	err = m.transaction(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, key, action)
		pipe.Expire(ctx, key, m.TTL)
		return nil
	})

	if err != nil {
		return "", err
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

	var res *redis.MapStringStringCmd

	err := m.transaction(ctx, func(pipe redis.Pipeliner) error {
		res = pipe.HGetAll(ctx, key)
		pipe.Del(ctx, key)
		return nil
	})

	if err != nil {
		return nil, err
	}

	if res.Err() != nil {
		return nil, res.Err()
	}

	var action A

	if len(res.Val()) == 0 {
		return nil, NonExistentAction
	}

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
