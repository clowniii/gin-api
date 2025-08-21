package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go-apiadmin/internal/domain/model"
	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/repository/dao"

	"go-apiadmin/internal/metrics"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type MenuService struct {
	DAO *dao.AdminMenuDAO
	// 使用统一 cache 接口；key 设计：
	// tree -> 菜单树 (keywords 为空)
	// access:uid:<id> -> 用户可访问菜单树
	Cache cache.Cache

	activeMux     sync.RWMutex    // 维护访问过 AccessMenu 的用户集合
	activeUID     map[int64]int64 // uid -> lastAccessUnix，用于精确失效 + 过期清理
	retention     time.Duration   // 记录保留时长
	maxActive     int             // 允许的最大活跃 UID 数
	lastCleanUnix int64           // 上次清理时间戳（秒）
	accessCount   uint64          // 累计访问计数，用于触发周期性清理
}

// tracer
func (s *MenuService) tracer() trace.Tracer { return otel.Tracer("service.menu") }

// NewMenuService 默认仅使用本地 simple cache
func NewMenuService(d *dao.AdminMenuDAO) *MenuService {
	return &MenuService{DAO: d, Cache: cache.NewSimpleAdapter(cache.New(120 * time.Second)), activeUID: make(map[int64]int64), retention: 30 * time.Minute, maxActive: 10000}
}

// NewMenuServiceWithCache 允许外部注入 LayeredCache
func NewMenuServiceWithCache(d *dao.AdminMenuDAO, c cache.Cache) *MenuService {
	return &MenuService{DAO: d, Cache: c, activeUID: make(map[int64]int64), retention: 30 * time.Minute, maxActive: 10000}
}

type ListMenuResult struct {
	List interface{} `json:"list"`
}

func (s *MenuService) List(ctx context.Context, keywords string) (*ListMenuResult, error) {
	ctx, span := s.tracer().Start(ctx, "MenuService.List")
	defer span.End()
	if keywords == "" { // 尝试缓存树
		if s.Cache != nil {
			if v, _ := s.Cache.Get(ctx, "menu:tree"); v != "" {
				if cache.IsNilSentinel(v) {
					metrics.CacheNilHit.Inc()
					return &ListMenuResult{List: []interface{}{}}, nil
				}
				return &ListMenuResult{List: dao.BuildTreeCachedJSON(v)}, nil
			}
		}
	}
	menus, err := s.DAO.ListMenus(ctx, keywords)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("list menus dao: %w", err)
	}
	if keywords != "" { // 扁平
		res := make([]map[string]interface{}, 0, len(menus))
		for _, m := range menus {
			res = append(res, map[string]interface{}{"id": m.ID, "fid": m.FID, "title": m.Title, "icon": m.Icon, "url": m.URL, "router": m.Router, "component": m.Component, "sort": m.Sort, "show": m.Show, "level": m.Level})
		}
		return &ListMenuResult{List: res}, nil
	}
	if len(menus) == 0 { // 空结果穿透保护
		if s.Cache != nil {
			_ = s.Cache.SetEX(ctx, "menu:tree", cache.WrapNil(true), 10*time.Second)
		}
		return &ListMenuResult{List: []interface{}{}}, nil
	}
	tree := dao.BuildTree(menus)
	if s.Cache != nil {
		b, _ := json.Marshal(tree)
		_ = s.Cache.SetEX(ctx, "menu:tree", string(b), 120*time.Second)
	}
	return &ListMenuResult{List: tree}, nil
}

type AddMenuParams struct {
	Fid                                 int64
	Title, Icon, URL, Router, Component string
	Sort, Show, Level                   int
}

func (s *MenuService) Add(ctx context.Context, p AddMenuParams) error {
	ctx, span := s.tracer().Start(ctx, "MenuService.Add")
	defer span.End()
	m := &model.AdminMenu{FID: p.Fid, Title: p.Title, Icon: p.Icon, URL: p.URL, Router: p.Router, Component: p.Component, Sort: p.Sort, Show: p.Show, Level: p.Level}
	err := s.DAO.Create(ctx, m)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("create menu: %w", err)
	}
	s.invalidate()
	return nil
}

type EditMenuParams struct {
	ID                                  int64
	Fid                                 *int64
	Title, Icon, URL, Router, Component *string
	Sort, Show, Level                   *int
}

func (s *MenuService) Edit(ctx context.Context, p EditMenuParams) error {
	ctx, span := s.tracer().Start(ctx, "MenuService.Edit")
	defer span.End()
	if p.ID <= 0 {
		return errors.New("invalid id")
	}
	menus, err := s.DAO.ListMenus(ctx, "")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("list menus for edit: %w", err)
	}
	var cur *model.AdminMenu
	for i := range menus {
		if menus[i].ID == p.ID {
			cur = &menus[i]
			break
		}
	}
	if cur == nil {
		return errors.New("not found")
	}
	if p.Fid != nil {
		cur.FID = *p.Fid
	}
	if p.Title != nil {
		cur.Title = *p.Title
	}
	if p.Icon != nil {
		cur.Icon = *p.Icon
	}
	if p.URL != nil {
		cur.URL = *p.URL
	}
	if p.Router != nil {
		cur.Router = *p.Router
	}
	if p.Component != nil {
		cur.Component = *p.Component
	}
	if p.Sort != nil {
		cur.Sort = *p.Sort
	}
	if p.Show != nil {
		cur.Show = *p.Show
	}
	if p.Level != nil {
		cur.Level = *p.Level
	}
	err = s.DAO.Update(ctx, cur)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("update menu: %w", err)
	}
	s.invalidate()
	return nil
}

func (s *MenuService) ChangeStatus(ctx context.Context, id int64, show int) error {
	ctx, span := s.tracer().Start(ctx, "MenuService.ChangeStatus")
	defer span.End()
	err := s.DAO.UpdateShow(ctx, id, show)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("update show: %w", err)
	}
	s.invalidate()
	return nil
}
func (s *MenuService) Delete(ctx context.Context, id int64) error {
	ctx, span := s.tracer().Start(ctx, "MenuService.Delete")
	defer span.End()
	err := s.DAO.Delete(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("delete menu: %w", err)
	}
	s.invalidate()
	return nil
}

// AccessMenu 缓存按用户
func (s *MenuService) AccessMenu(ctx context.Context, uid int64) ([]map[string]interface{}, error) {
	ctx, span := s.tracer().Start(ctx, "MenuService.AccessMenu")
	defer span.End()
	// 记录活跃 UID（懒触发清理）
	now := time.Now().Unix()
	s.activeMux.Lock()
	s.activeUID[uid] = now
	s.activeMux.Unlock()
	s.maybeCleanup(now)
	if s.Cache != nil {
		if v, _ := s.Cache.Get(ctx, s.uidKey(uid)); v != "" {
			if cache.IsNilSentinel(v) {
				metrics.CacheNilHit.Inc()
				return []map[string]interface{}{}, nil
			}
			var tree []map[string]interface{}
			if err := json.Unmarshal([]byte(v), &tree); err == nil && len(tree) > 0 {
				return tree, nil
			}
		}
	}
	menus, err := s.DAO.ListMenus(ctx, "")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("list menus for access: %w", err)
	}
	if len(menus) == 0 {
		if s.Cache != nil {
			_ = s.Cache.SetEX(ctx, s.uidKey(uid), cache.WrapNil(true), 10*time.Second)
		}
		return []map[string]interface{}{}, nil
	}
	tree := dao.BuildTree(menus)
	if s.Cache != nil {
		b, _ := json.Marshal(tree)
		_ = s.Cache.SetEX(ctx, s.uidKey(uid), string(b), 120*time.Second)
	}
	return tree, nil
}

// maybeCleanup 懒触发活跃 UID 清理：
// 条件：距离上次清理 >= 5 分钟 或 activeUID 长度 > maxActive*2 或 访问计数增量到达阈值（1000）
func (s *MenuService) maybeCleanup(now int64) {
	ac := atomic.AddUint64(&s.accessCount, 1)
	if ac%1000 != 0 { // 减少判定频率
		return
	}
	last := atomic.LoadInt64(&s.lastCleanUnix)
	need := false
	if now-last >= 300 { // 5 分钟
		need = true
	}
	s.activeMux.RLock()
	sz := len(s.activeUID)
	s.activeMux.RUnlock()
	if sz > s.maxActive*2 {
		need = true
	}
	if !need {
		return
	}
	if !atomic.CompareAndSwapInt64(&s.lastCleanUnix, last, now) {
		return // 其他协程已开始
	}
	// 执行清理
	cutoff := now - int64(s.retention.Seconds())
	toDelete := make([]int64, 0)
	s.activeMux.RLock()
	for uid, ts := range s.activeUID {
		if ts < cutoff { // 过期
			toDelete = append(toDelete, uid)
		}
	}
	// 如果仍超容量，收集最旧若干
	if len(s.activeUID)-len(toDelete) > s.maxActive {
		// 取所有剩余，排序按 ts 升序截断
		remain := make([]struct {
			uid int64
			ts  int64
		}, 0, len(s.activeUID))
		for uid, ts := range s.activeUID {
			if ts >= cutoff { // 未过期
				remain = append(remain, struct {
					uid int64
					ts  int64
				}{uid: uid, ts: ts})
			}
		}
		// 简易 O(n^2) 插入排序（数量有限），避免引入新依赖
		for i := 1; i < len(remain); i++ {
			j := i
			for j > 0 && remain[j].ts < remain[j-1].ts {
				remain[j], remain[j-1] = remain[j-1], remain[j]
				j--
			}
		}
		needRemove := len(remain) - s.maxActive
		for i := 0; i < needRemove && i < len(remain); i++ {
			toDelete = append(toDelete, remain[i].uid)
		}
	}
	s.activeMux.RUnlock()
	if len(toDelete) == 0 {
		return
	}
	s.activeMux.Lock()
	for _, uid := range toDelete {
		delete(s.activeUID, uid)
	}
	s.activeMux.Unlock()
}

// ========== 缓存内部方法 ==========
func (s *MenuService) invalidate() {
	if s.Cache != nil {
		_ = s.Cache.Del(context.Background(), "menu:tree")
		// 精确失效所有活跃用户 access 缓存
		s.activeMux.RLock()
		uids := make([]int64, 0, len(s.activeUID))
		for id := range s.activeUID {
			uids = append(uids, id)
		}
		s.activeMux.RUnlock()
		if len(uids) > 0 {
			keys := make([]string, 0, len(uids))
			for _, id := range uids {
				keys = append(keys, s.uidKey(id))
			}
			_ = s.Cache.Del(context.Background(), keys...)
		}
	}
}
func (s *MenuService) uidKey(uid int64) string { return "access:uid:" + _intToStr(uid) }

// 简单的更新时间方法占位
func (s *MenuService) Touch(_ context.Context) { _ = time.Now() }
