package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/repository/dao"
	redisrepo "go-apiadmin/internal/repository/redis"
)

// PermissionService 负责用户权限加载与缓存（本地内存 + Redis 持久 或 统一 LayeredCache）
// 规则：admin_auth_rule.url (不含域名) 作为权限资源标识，统一转为小写
// 简化：method 不区分，如需区分可在规则表增加字段再扩展

type PermissionService struct {
	Groups      *dao.AdminAuthGroupAccessDAO
	Rules       *dao.AdminAuthRuleDAO
	UserDAO     *dao.AdminUserDAO
	Redis       *redisrepo.Client       // 旧实现中 L2 Redis；保留以兼容旧逻辑
	cacheMux    sync.RWMutex            // 旧实现 L1 本地 map
	cache       map[int64]permCacheItem // 旧实现本地 map
	ttl         time.Duration
	redisPrefix string
	Cache       cache.Cache // 新增: 统一缓存接口 (可为 LayeredCache)，若设置则优先使用，不再走旧 map+redis 流程

	// metrics
	metricUnifiedHit uint64 // 统一缓存命中
	metricLocalHit   uint64 // 旧本地 map 命中
	metricRedisHit   uint64 // 旧 redis 命中
	metricDBLoad     uint64 // DB 回源次数
}

type permCacheItem struct {
	Expires time.Time
	Set     map[string]struct{}
}

func NewPermissionService(gr *dao.AdminAuthGroupAccessDAO, rule *dao.AdminAuthRuleDAO, u *dao.AdminUserDAO, r *redisrepo.Client) *PermissionService {
	return &PermissionService{Groups: gr, Rules: rule, UserDAO: u, Redis: r, cache: make(map[int64]permCacheItem), ttl: 5 * time.Minute, redisPrefix: "perm:user:"}
}

// NewPermissionServiceWithCache 使用统一 cache（推荐，支持 LayeredCache）
// r 仍然可传入以兼容旧逻辑或供其它场景使用；如果不需要可传 nil。
func NewPermissionServiceWithCache(gr *dao.AdminAuthGroupAccessDAO, rule *dao.AdminAuthRuleDAO, u *dao.AdminUserDAO, r *redisrepo.Client, c cache.Cache) *PermissionService {
	ps := NewPermissionService(gr, rule, u, r)
	ps.Cache = c
	return ps
}

// GetUserPermissions 返回用户权限集合
// 优先使用统一 Cache；若未注入 Cache 则退回旧 (L1 map + Redis + DB) 逻辑
func (p *PermissionService) GetUserPermissions(ctx context.Context, uid int64) (map[string]struct{}, error) {
	if p.Cache != nil { // 统一缓存路径
		key := p.redisKey(uid)
		if v, _ := p.Cache.Get(ctx, key); v != "" {
			var arr []string
			if json.Unmarshal([]byte(v), &arr) == nil {
				set := make(map[string]struct{}, len(arr))
				for _, s := range arr {
					set[s] = struct{}{}
				}
				atomic.AddUint64(&p.metricUnifiedHit, 1)
				return set, nil
			}
		}
		// miss -> DB 回源
		gids, err := p.Groups.ListGroupIDsByUser(ctx, uid)
		if err != nil {
			return nil, err
		}
		rules, err := p.Rules.ListByGroupIDs(ctx, gids)
		if err != nil {
			return nil, err
		}
		set := make(map[string]struct{}, len(rules))
		arr := make([]string, 0, len(rules))
		for _, r := range rules {
			keyURL := strings.ToLower(r.URL)
			set[keyURL] = struct{}{}
			arr = append(arr, keyURL)
		}
		if b, err := json.Marshal(arr); err == nil {
			_ = p.Cache.SetEX(ctx, key, string(b), p.ttl)
		}
		atomic.AddUint64(&p.metricDBLoad, 1)
		return set, nil
	}

	// ===== 旧实现 (本地 map + Redis) 保留兼容 =====
	// 1. 进程内缓存
	p.cacheMux.RLock()
	item, ok := p.cache[uid]
	if ok && time.Now().Before(item.Expires) {
		defer p.cacheMux.RUnlock()
		atomic.AddUint64(&p.metricLocalHit, 1)
		return item.Set, nil
	}
	p.cacheMux.RUnlock()

	// 2. Redis 缓存
	if p.Redis != nil {
		if b, err := p.Redis.Client.Get(ctx, p.redisKey(uid)).Bytes(); err == nil && len(b) > 0 {
			var arr []string
			if json.Unmarshal(b, &arr) == nil {
				set := make(map[string]struct{}, len(arr))
				for _, s := range arr {
					set[s] = struct{}{}
				}
				p.cacheMux.Lock()
				p.cache[uid] = permCacheItem{Expires: time.Now().Add(p.ttl / 2), Set: set}
				p.cacheMux.Unlock()
				atomic.AddUint64(&p.metricRedisHit, 1)
				return set, nil
			}
		}
	}

	// 3. DB 回源
	gids, err := p.Groups.ListGroupIDsByUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	rules, err := p.Rules.ListByGroupIDs(ctx, gids)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(rules))
	arr := make([]string, 0, len(rules))
	for _, r := range rules {
		key := strings.ToLower(r.URL)
		set[key] = struct{}{}
		arr = append(arr, key)
	}
	p.cacheMux.Lock()
	p.cache[uid] = permCacheItem{Expires: time.Now().Add(p.ttl), Set: set}
	p.cacheMux.Unlock()
	if p.Redis != nil {
		if b, err := json.Marshal(arr); err == nil {
			_ = p.Redis.SetTTL(ctx, p.redisKey(uid), b, p.ttl)
		}
	}
	atomic.AddUint64(&p.metricDBLoad, 1)
	return set, nil
}

// Invalidate 清除用户缓存（组或规则变化后调用）
func (p *PermissionService) Invalidate(uid int64) {
	if p.Cache != nil { // 新缓存
		_ = p.Cache.Del(context.Background(), p.redisKey(uid))
		return
	}
	p.cacheMux.Lock()
	delete(p.cache, uid)
	p.cacheMux.Unlock()
	if p.Redis != nil {
		_ = p.Redis.Client.Del(context.Background(), p.redisKey(uid)).Err()
	}
}

// InvalidateUsersByGroup 依据组使相关用户权限失效
func (p *PermissionService) InvalidateUsersByGroup(ctx context.Context, gid int64) {
	uids, err := p.Groups.ListUserIDsByGroup(ctx, gid)
	if err != nil {
		return
	}
	if p.Cache != nil {
		for _, uid := range uids {
			_ = p.Cache.Del(context.Background(), p.redisKey(uid))
		}
		return
	}
	p.cacheMux.Lock()
	for _, uid := range uids {
		delete(p.cache, uid)
	}
	p.cacheMux.Unlock()
	if p.Redis != nil {
		for _, uid := range uids {
			_ = p.Redis.Client.Del(context.Background(), p.redisKey(uid)).Err()
		}
	}
}

// InvalidateAll 全量失效（例如批量脚本后）
func (p *PermissionService) InvalidateAll() {
	if p.Cache != nil {
		// 只能逐个 key；调用方如需全量可重建实例或使用外层前缀刷新
		return
	}
	p.cacheMux.Lock()
	p.cache = make(map[int64]permCacheItem)
	p.cacheMux.Unlock()
	// Redis 批量删除（简单遍历 scan）可后续实现，这里暂不实现避免引入复杂度
}

// HasPermission 判断用户是否拥有指定URL权限（path 已规范化）
func (p *PermissionService) HasPermission(ctx context.Context, uid int64, path string) bool {
	perms, err := p.GetUserPermissions(ctx, uid)
	if err != nil {
		return false
	}
	path = strings.ToLower(path)
	_, ok := perms[path]
	return ok
}

func (p *PermissionService) redisKey(uid int64) string {
	return p.redisPrefix + strconv.FormatInt(uid, 10)
}

// PermissionMetrics 指标快照
// HitRate = (unifiedHit + localHit + redisHit) / totalRequests
// totalRequests = unifiedHit + localHit + redisHit + dbLoad
// 若 total=0 则 HitRate=0

type PermissionMetrics struct {
	UnifiedHit uint64  `json:"unified_hit"`
	LocalHit   uint64  `json:"local_hit"`
	RedisHit   uint64  `json:"redis_hit"`
	DBLoad     uint64  `json:"db_load"`
	HitRate    float64 `json:"hit_rate"`
}

func (p *PermissionService) SnapshotMetrics() PermissionMetrics {
	uh := atomic.LoadUint64(&p.metricUnifiedHit)
	lh := atomic.LoadUint64(&p.metricLocalHit)
	rh := atomic.LoadUint64(&p.metricRedisHit)
	db := atomic.LoadUint64(&p.metricDBLoad)
	total := uh + lh + rh + db
	rate := 0.0
	if total > 0 {
		rate = float64(uh+lh+rh) / float64(total)
	}
	return PermissionMetrics{UnifiedHit: uh, LocalHit: lh, RedisHit: rh, DBLoad: db, HitRate: rate}
}
func (p *PermissionService) ResetMetrics() {
	atomic.StoreUint64(&p.metricUnifiedHit, 0)
	atomic.StoreUint64(&p.metricLocalHit, 0)
	atomic.StoreUint64(&p.metricRedisHit, 0)
	atomic.StoreUint64(&p.metricDBLoad, 0)
}
