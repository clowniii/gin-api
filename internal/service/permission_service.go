package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/repository/dao"
	redisrepo "go-apiadmin/internal/repository/redis"

	"go-apiadmin/internal/metrics"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// PermissionService 负责用户权限加载与缓存（本地内存 + Redis 持久 或 统一 LayeredCache）
// 规则：admin_auth_rule.url (不含域名) 作为权限资源标识，统一转为小写
// 简化：method 不区分，如需区分可在规则表增加字段再扩展

type PermissionService struct {
	Groups      *dao.AdminAuthGroupAccessDAO
	Rules       *dao.AdminAuthRuleDAO
	UserDAO     *dao.AdminUserDAO
	MenuDAO     *dao.AdminMenuDAO
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

func NewPermissionService(gr *dao.AdminAuthGroupAccessDAO, rule *dao.AdminAuthRuleDAO, u *dao.AdminUserDAO, m *dao.AdminMenuDAO, r *redisrepo.Client) *PermissionService {
	return &PermissionService{Groups: gr, Rules: rule, UserDAO: u, MenuDAO: m, Redis: r, cache: make(map[int64]permCacheItem), ttl: 5 * time.Minute, redisPrefix: "perm:user:"}
}

// NewPermissionServiceWithCache 使用统一 cache（推荐，支持 LayeredCache）
// r 仍然可传入以兼容旧逻辑或供其它场景使用；如果不需要可传 nil。
func NewPermissionServiceWithCache(gr *dao.AdminAuthGroupAccessDAO, rule *dao.AdminAuthRuleDAO, u *dao.AdminUserDAO, m *dao.AdminMenuDAO, r *redisrepo.Client, c cache.Cache) *PermissionService {
	ps := NewPermissionService(gr, rule, u, m, r)
	ps.Cache = c
	return ps
}

// tracer 获取 service tracer
func (p *PermissionService) tracer() trace.Tracer { return otel.Tracer("service.permission") }

// SetCacheWithTTL 封装: 统一 SetEX 使用 JitterTTL
func (p *PermissionService) setCacheWithTTL(ctx context.Context, key string, val string, ttl time.Duration) {
	if p.Cache != nil {
		_ = p.Cache.SetEX(ctx, key, val, cache.JitterTTL(ttl))
	}
}

// GetUserPermissions 返回用户权限集合
// 优先使用统一 Cache；若未注入 Cache 则退回旧 (L1 map + Redis + DB) 逻辑
func (p *PermissionService) GetUserPermissions(ctx context.Context, uid int64) (map[string]struct{}, error) {
	ctx, span := p.tracer().Start(ctx, "PermissionService.GetUserPermissions", trace.WithAttributes())
	defer span.End()
	if p.Cache != nil { // 统一缓存路径
		if uid == 1 { // 超级管理员: 直接基于全部菜单 URL 构建权限集合（不受 show 限制）
			key := p.redisKey(uid)
			if v, _ := p.Cache.Get(ctx, key); v != "" { // 尝试命中缓存
				if cache.IsNilSentinel(v) {
					atomic.AddUint64(&p.metricUnifiedHit, 1)
					return map[string]struct{}{}, nil
				}
				var arr []string
				if json.Unmarshal([]byte(v), &arr) == nil && len(arr) > 0 {
					set := make(map[string]struct{}, len(arr))
					for _, s := range arr {
						set[s] = struct{}{}
					}
					atomic.AddUint64(&p.metricUnifiedHit, 1)
					return set, nil
				}
			}
			// miss -> 加载全部菜单
			menus, err := p.MenuDAO.ListMenus(ctx, "")
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, fmt.Errorf("list all menus for super admin: %w", err)
			}
			set := make(map[string]struct{}, len(menus))
			arr := make([]string, 0, len(menus))
			for _, m := range menus { // 不限 show，完整权限
				u := strings.ToLower(m.URL)
				if u == "" { // 跳过空 URL（与旧逻辑保持一致，可按需保留）
					continue
				}
				set[u] = struct{}{}
				arr = append(arr, u)
			}
			if len(arr) == 0 { // 空 sentinel 防穿透
				p.setCacheWithTTL(ctx, key, cache.WrapNil(true), 30*time.Second)
			} else if b, err := json.Marshal(arr); err == nil {
				p.setCacheWithTTL(ctx, key, string(b), p.ttl)
			}
			atomic.AddUint64(&p.metricDBLoad, 1)
			return set, nil
		}
		// 非超级管理员
		key := p.redisKey(uid)
		if v, _ := p.Cache.Get(ctx, key); v != "" {
			if cache.IsNilSentinel(v) {
				atomic.AddUint64(&p.metricUnifiedHit, 1)
				return map[string]struct{}{}, nil
			}
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
		// miss -> 根据用户组与规则过滤菜单(show=1且规则 URL 集合里)
		gids, err := p.Groups.ListGroupIDsByUser(ctx, uid)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("get user group ids: %w", err)
		}
		if len(gids) == 0 { // 无分组 -> 空权限
			p.setCacheWithTTL(ctx, key, cache.WrapNil(true), 15*time.Second)
			return map[string]struct{}{}, nil
		}
		rules, err := p.Rules.ListByGroupIDs(ctx, gids)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("list rules by group ids: %w", err)
		}
		if len(rules) == 0 {
			p.setCacheWithTTL(ctx, key, cache.WrapNil(true), 15*time.Second)
			atomic.AddUint64(&p.metricDBLoad, 1)
			return map[string]struct{}{}, nil
		}
		// 构建规则 URL set
		ruleSet := make(map[string]struct{}, len(rules))
		for _, r := range rules {
			if r.URL == "" { // 空 URL 仍然加入（兼容旧逻辑 access[""]）
				continue
			}
			ruleSet[strings.ToLower(r.URL)] = struct{}{}
		}
		// 加载菜单并过滤 show=1 且 URL 在 ruleSet 中
		menus, err := p.MenuDAO.ListMenus(ctx, "")
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("list menus for user: %w", err)
		}
		set := make(map[string]struct{})
		arr := make([]string, 0)
		for _, m := range menus {
			if m.Show != 1 { // 仅显示状态菜单
				continue
			}
			u := strings.ToLower(m.URL)
			if u == "" { // 空 URL 允许加入一次（与旧逻辑 access[""] = struct{}{} 类似，可选）
				continue
			}
			if _, ok := ruleSet[u]; ok {
				set[u] = struct{}{}
				arr = append(arr, u)
			}
		}
		if len(arr) == 0 { // 没有匹配菜单
			p.setCacheWithTTL(ctx, key, cache.WrapNil(true), 15*time.Second)
			atomic.AddUint64(&p.metricDBLoad, 1)
			return map[string]struct{}{}, nil
		}
		if b, err := json.Marshal(arr); err == nil {
			p.setCacheWithTTL(ctx, key, string(b), p.ttl)
		}
		atomic.AddUint64(&p.metricDBLoad, 1)
		return set, nil
	}

	// ===== 旧实现 (本地 map + Redis) 保留兼容 =====
	// 超级管理员: 直接加载全部启用规则
	if uid == 1 {
		rules, err := p.Rules.List(ctx, nil)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("list all rules for super admin (legacy path): %w", err)
		}
		set := make(map[string]struct{}, len(rules))
		for _, r := range rules {
			if r.Status != 1 {
				continue
			}
			set[strings.ToLower(r.URL)] = struct{}{}
		}
		atomic.AddUint64(&p.metricDBLoad, 1)
		return set, nil
	}
	// 1. 进程内缓存 (不做 sentinel，这里仅适用非空结果缓存)
	p.cacheMux.RLock()
	item, ok := p.cache[uid]
	if ok && time.Now().Before(item.Expires) {
		defer p.cacheMux.RUnlock()
		atomic.AddUint64(&p.metricLocalHit, 1)
		return item.Set, nil
	}
	p.cacheMux.RUnlock()

	// 2. Redis 缓存（老逻辑也加入 sentinel 支持）
	if p.Redis != nil {
		if b, err := p.Redis.Client.Get(ctx, p.redisKey(uid)).Bytes(); err == nil && len(b) > 0 {
			if string(b) == cache.WrapNil(true) { // sentinel
				atomic.AddUint64(&p.metricRedisHit, 1)
				return map[string]struct{}{}, nil
			}
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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("get user group ids (fallback): %w", err)
	}
	rules, err := p.Rules.ListByGroupIDs(ctx, gids)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("list rules by group ids (fallback): %w", err)
	}
	if len(rules) == 0 { // 空 sentinel
		if p.Redis != nil {
			_ = p.Redis.SetTTL(ctx, p.redisKey(uid), []byte(cache.WrapNil(true)), 15*time.Second)
		}
		atomic.AddUint64(&p.metricDBLoad, 1)
		return map[string]struct{}{}, nil
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
	ctx, span := p.tracer().Start(context.Background(), "PermissionService.Invalidate")
	defer span.End()
	metrics.PermissionInvalidateTotal.WithLabelValues("single").Inc()
	if p.Cache != nil { // 新缓存
		_ = p.Cache.Del(ctx, p.redisKey(uid))
		return
	}
	p.cacheMux.Lock()
	delete(p.cache, uid)
	p.cacheMux.Unlock()
	if p.Redis != nil {
		_ = p.Redis.Client.Del(ctx, p.redisKey(uid)).Err()
	}
}

// InvalidateUsersByGroup 依据组使相关用户权限失效
func (p *PermissionService) InvalidateUsersByGroup(ctx context.Context, gid int64) {
	ctx, span := p.tracer().Start(ctx, "PermissionService.InvalidateUsersByGroup")
	defer span.End()
	uids, err := p.Groups.ListUserIDsByGroup(ctx, gid)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}
	metrics.PermissionInvalidateTotal.WithLabelValues("group").Inc()
	if len(uids) > 0 {
		metrics.PermissionInvalidateUsersTotal.Add(float64(len(uids)))
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
	_, span := p.tracer().Start(context.Background(), "PermissionService.InvalidateAll")
	defer span.End()
	metrics.PermissionInvalidateTotal.WithLabelValues("all").Inc()
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
	ctx, span := p.tracer().Start(ctx, "PermissionService.HasPermission")
	defer span.End()
	perms, err := p.GetUserPermissions(ctx, uid)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
