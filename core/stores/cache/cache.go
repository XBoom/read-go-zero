package cache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/zeromicro/go-zero/core/errorx"
	"github.com/zeromicro/go-zero/core/hash"
	"github.com/zeromicro/go-zero/core/syncx"
)

type (
	// Cache interface 定义缓存接口
	Cache interface {
		// Del 根据Keys删除(支持批量删除)
		Del(keys ...string) error
		// DelCtx 根据Keys删除(支持批量删除 ctx)
		DelCtx(ctx context.Context, keys ...string) error
		// Get 根据 Key 获取缓存存入 val
		Get(key string, val interface{}) error
		// GetCtx 根据 Key 获取缓存存入 val
		GetCtx(ctx context.Context, key string, val interface{}) error
		// IsNotFound 判断是否是缓存没有找到的错误 errNotFound.
		IsNotFound(err error) bool
		// Set 设置缓存 k/v
		Set(key string, val interface{}) error
		// SetCtx 设置缓存 k/v
		SetCtx(ctx context.Context, key string, val interface{}) error
		// SetWithExpire 设置缓存 k/v,带过期时间
		SetWithExpire(key string, val interface{}, expire time.Duration) error
		// SetWithExpireCtx 设置缓存 k/v,带过期时间
		SetWithExpireCtx(ctx context.Context, key string, val interface{}, expire time.Duration) error
		// Take 首先从缓存获取，如果缓存没有则从数据库加载并设置缓存
		Take(val interface{}, key string, query func(val interface{}) error) error
		// TakeCtx 首先从缓存获取，如果缓存没有则从数据库加载并设置缓存
		TakeCtx(ctx context.Context, val interface{}, key string, query func(val interface{}) error) error
		// TakeWithExpire 首先从缓存获取，如果缓存没有则从数据库加载并设置缓存,缓存过期时间为 expire
		TakeWithExpire(val interface{}, key string, query func(val interface{}, expire time.Duration) error) error
		// TakeWithExpireCtx 首先从缓存获取，如果缓存没有则从数据库加载并设置缓存,缓存过期时间为 expire
		TakeWithExpireCtx(ctx context.Context, val interface{}, key string,
			query func(val interface{}, expire time.Duration) error) error
	}

	//缓存集群
	cacheCluster struct {
		dispatcher  *hash.ConsistentHash
		errNotFound error
	}
)

// New returns a Cache.
func New(c ClusterConf, barrier syncx.SingleFlight, st *Stat, errNotFound error,
	opts ...Option) Cache {
	if len(c) == 0 || TotalWeights(c) <= 0 {
		log.Fatal("no cache nodes")
	}

	if len(c) == 1 {
		return NewNode(c[0].NewRedis(), barrier, st, errNotFound, opts...)
	}

	dispatcher := hash.NewConsistentHash()
	for _, node := range c {
		cn := NewNode(node.NewRedis(), barrier, st, errNotFound, opts...)
		dispatcher.AddWithWeight(cn, node.Weight)
	}

	return cacheCluster{
		dispatcher:  dispatcher,
		errNotFound: errNotFound,
	}
}

// Del deletes cached values with keys.
func (cc cacheCluster) Del(keys ...string) error {
	return cc.DelCtx(context.Background(), keys...)
}

// DelCtx deletes cached values with keys.
func (cc cacheCluster) DelCtx(ctx context.Context, keys ...string) error {
	switch len(keys) {
	case 0:
		return nil
	case 1:
		key := keys[0]
		c, ok := cc.dispatcher.Get(key)
		if !ok {
			return cc.errNotFound
		}

		return c.(Cache).DelCtx(ctx, key)
	default:
		var be errorx.BatchError
		nodes := make(map[interface{}][]string)
		for _, key := range keys {
			c, ok := cc.dispatcher.Get(key)
			if !ok {
				be.Add(fmt.Errorf("key %q not found", key))
				continue
			}

			nodes[c] = append(nodes[c], key)
		}
		for c, ks := range nodes {
			if err := c.(Cache).DelCtx(ctx, ks...); err != nil {
				be.Add(err)
			}
		}

		return be.Err()
	}
}

// Get gets the cache with key and fills into v.
func (cc cacheCluster) Get(key string, val interface{}) error {
	return cc.GetCtx(context.Background(), key, val)
}

// GetCtx gets the cache with key and fills into v.
func (cc cacheCluster) GetCtx(ctx context.Context, key string, val interface{}) error {
	c, ok := cc.dispatcher.Get(key)
	if !ok {
		return cc.errNotFound
	}

	return c.(Cache).GetCtx(ctx, key, val)
}

// IsNotFound checks if the given error is the defined errNotFound.
func (cc cacheCluster) IsNotFound(err error) bool {
	return errors.Is(err, cc.errNotFound)
}

// Set sets the cache with key and v, using c.expiry.
func (cc cacheCluster) Set(key string, val interface{}) error {
	return cc.SetCtx(context.Background(), key, val)
}

// SetCtx sets the cache with key and v, using c.expiry.
func (cc cacheCluster) SetCtx(ctx context.Context, key string, val interface{}) error {
	c, ok := cc.dispatcher.Get(key)
	if !ok {
		return cc.errNotFound
	}

	return c.(Cache).SetCtx(ctx, key, val)
}

// SetWithExpire sets the cache with key and v, using given expire.
func (cc cacheCluster) SetWithExpire(key string, val interface{}, expire time.Duration) error {
	return cc.SetWithExpireCtx(context.Background(), key, val, expire)
}

// SetWithExpireCtx sets the cache with key and v, using given expire.
func (cc cacheCluster) SetWithExpireCtx(ctx context.Context, key string, val interface{}, expire time.Duration) error {
	c, ok := cc.dispatcher.Get(key)
	if !ok {
		return cc.errNotFound
	}

	return c.(Cache).SetWithExpireCtx(ctx, key, val, expire)
}

// Take takes the result from cache first, if not found,
// query from DB and set cache using c.expiry, then return the result.
func (cc cacheCluster) Take(val interface{}, key string, query func(val interface{}) error) error {
	return cc.TakeCtx(context.Background(), val, key, query)
}

// TakeCtx takes the result from cache first, if not found,
// query from DB and set cache using c.expiry, then return the result.
func (cc cacheCluster) TakeCtx(ctx context.Context, val interface{}, key string, query func(val interface{}) error) error {
	c, ok := cc.dispatcher.Get(key)
	if !ok {
		return cc.errNotFound
	}

	return c.(Cache).TakeCtx(ctx, val, key, query)
}

// TakeWithExpire takes the result from cache first, if not found,
// query from DB and set cache using given expire, then return the result.
func (cc cacheCluster) TakeWithExpire(val interface{}, key string, query func(val interface{}, expire time.Duration) error) error {
	return cc.TakeWithExpireCtx(context.Background(), val, key, query)
}

// TakeWithExpireCtx takes the result from cache first, if not found,
// query from DB and set cache using given expire, then return the result.
func (cc cacheCluster) TakeWithExpireCtx(ctx context.Context, val interface{}, key string, query func(val interface{}, expire time.Duration) error) error {
	c, ok := cc.dispatcher.Get(key)
	if !ok {
		return cc.errNotFound
	}

	return c.(Cache).TakeWithExpireCtx(ctx, val, key, query)
}
