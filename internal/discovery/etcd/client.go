package etcd

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Config struct {
	Endpoints []string
	TTL       int
}

type Client struct{ *clientv3.Client }

func New(cfg Config) (*Client, error) {
	cli, err := clientv3.New(clientv3.Config{Endpoints: cfg.Endpoints, DialTimeout: 5 * time.Second})
	if err != nil {
		return nil, err
	}
	return &Client{cli}, nil
}

// Register 返回 leaseID 以便优雅下线时主动撤销
func (c *Client) Register(ctx context.Context, key, val string, ttl int64) (clientv3.LeaseID, error) {
	lease, err := c.Client.Grant(ctx, ttl)
	if err != nil {
		return 0, err
	}
	_, err = c.Client.Put(ctx, key, val, clientv3.WithLease(lease.ID))
	if err != nil {
		return 0, err
	}
	ch, kaErr := c.Client.KeepAlive(ctx, lease.ID)
	if kaErr != nil {
		return 0, kaErr
	}
	go func() {
		for range ch { // 消耗 keepalive channel 维持租约
		}
	}()
	return lease.ID, nil
}

// Deregister 主动删除 key 并撤销租约
func (c *Client) Deregister(ctx context.Context, key string, leaseID clientv3.LeaseID) error {
	// 删除 key (可能已经过期，无需强制处理错误)
	_, _ = c.Client.Delete(ctx, key)
	if leaseID > 0 {
		_, _ = c.Client.Revoke(ctx, leaseID)
	}
	return nil
}

func (c *Client) Discover(ctx context.Context, prefix string) (map[string]string, error) {
	resp, err := c.Client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for _, kv := range resp.Kvs {
		m[string(kv.Key)] = string(kv.Value)
	}
	return m, nil
}

func (c *Client) Close() error { return c.Client.Close() }
