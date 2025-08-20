package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go-apiadmin/internal/domain/model"
	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/repository/dao"
)

type MenuService struct {
	DAO *dao.AdminMenuDAO
	// 使用统一 cache 接口；key 设计：
	// tree -> 菜单树 (keywords 为空)
	// access:uid:<id> -> 用户可访问菜单树
	Cache cache.Cache
	// 仍保留一个本地简单缓存用于降级（可选）
}

// NewMenuService 默认仅使用本地 simple cache
func NewMenuService(d *dao.AdminMenuDAO) *MenuService {
	return &MenuService{DAO: d, Cache: cache.NewSimpleAdapter(cache.New(120 * time.Second))}
}

// NewMenuServiceWithCache 允许外部注入 LayeredCache
func NewMenuServiceWithCache(d *dao.AdminMenuDAO, c cache.Cache) *MenuService {
	return &MenuService{DAO: d, Cache: c}
}

type ListMenuResult struct {
	List interface{} `json:"list"`
}

func (s *MenuService) List(ctx context.Context, keywords string) (*ListMenuResult, error) {
	if keywords == "" { // 尝试缓存树
		if s.Cache != nil {
			if v, _ := s.Cache.Get(ctx, "menu:tree"); v != "" {
				return &ListMenuResult{List: dao.BuildTreeCachedJSON(v)}, nil
			}
		}
	}
	menus, err := s.DAO.ListMenus(ctx, keywords)
	if err != nil {
		return nil, err
	}
	if keywords != "" { // 扁平
		res := make([]map[string]interface{}, 0, len(menus))
		for _, m := range menus {
			res = append(res, map[string]interface{}{"id": m.ID, "fid": m.FID, "title": m.Title, "icon": m.Icon, "url": m.URL, "router": m.Router, "component": m.Component, "sort": m.Sort, "show": m.Show, "level": m.Level})
		}
		return &ListMenuResult{List: res}, nil
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
	m := &model.AdminMenu{FID: p.Fid, Title: p.Title, Icon: p.Icon, URL: p.URL, Router: p.Router, Component: p.Component, Sort: p.Sort, Show: p.Show, Level: p.Level}
	err := s.DAO.Create(ctx, m)
	if err == nil {
		s.invalidate()
	}
	return err
}

type EditMenuParams struct {
	ID                                  int64
	Fid                                 *int64
	Title, Icon, URL, Router, Component *string
	Sort, Show, Level                   *int
}

func (s *MenuService) Edit(ctx context.Context, p EditMenuParams) error {
	if p.ID <= 0 {
		return errors.New("invalid id")
	}
	menus, err := s.DAO.ListMenus(ctx, "")
	if err != nil {
		return err
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
	if err == nil {
		s.invalidate()
	}
	return err
}

func (s *MenuService) ChangeStatus(ctx context.Context, id int64, show int) error {
	err := s.DAO.UpdateShow(ctx, id, show)
	if err == nil {
		s.invalidate()
	}
	return err
}
func (s *MenuService) Delete(ctx context.Context, id int64) error {
	err := s.DAO.Delete(ctx, id)
	if err == nil {
		s.invalidate()
	}
	return err
}

// AccessMenu 缓存按用户
func (s *MenuService) AccessMenu(ctx context.Context, uid int64) ([]map[string]interface{}, error) {
	if s.Cache != nil {
		if v, _ := s.Cache.Get(ctx, s.uidKey(uid)); v != "" {
			var tree []map[string]interface{}
			if err := json.Unmarshal([]byte(v), &tree); err == nil && len(tree) > 0 {
				return tree, nil
			}
		}
	}
	menus, err := s.DAO.ListMenus(ctx, "")
	if err != nil {
		return nil, err
	}
	tree := dao.BuildTree(menus)
	if s.Cache != nil {
		b, _ := json.Marshal(tree)
		_ = s.Cache.SetEX(ctx, s.uidKey(uid), string(b), 120*time.Second)
	}
	return tree, nil
}

// ========== 缓存内部方法 ==========
func (s *MenuService) invalidate() {
	if s.Cache != nil {
		_ = s.Cache.Del(context.Background(), "menu:tree")
		// 用户 access 缓存精确删除需追踪 keys，这里简化为无操作；可选：维护 uid 列表，再逐个删。
	}
}
func (s *MenuService) uidKey(uid int64) string { return "access:uid:" + _intToStr(uid) }

// 简单的更新时间方法占位
func (s *MenuService) Touch(_ context.Context) { _ = time.Now() }
