package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go-apiadmin/internal/domain/model"
	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/repository/dao"

	"go-apiadmin/internal/metrics"
)

type AuthGroupService struct {
	Groups *dao.AdminAuthGroupDAO
	Rel    *dao.AdminAuthGroupAccessDAO
	Perm   *PermissionService
	Cache  cache.Cache // 新增: 列表/单对象缓存（当前只缓存列表）
}

func NewAuthGroupService(g *dao.AdminAuthGroupDAO, rel *dao.AdminAuthGroupAccessDAO, perm *PermissionService) *AuthGroupService {
	return &AuthGroupService{Groups: g, Rel: rel, Perm: perm, Cache: cache.NewSimpleAdapter(cache.New(60 * time.Second))}
}

// NewAuthGroupServiceWithCache 允许外部注入 layered cache
func NewAuthGroupServiceWithCache(g *dao.AdminAuthGroupDAO, rel *dao.AdminAuthGroupAccessDAO, perm *PermissionService, c cache.Cache) *AuthGroupService {
	return &AuthGroupService{Groups: g, Rel: rel, Perm: perm, Cache: c}
}

type GroupDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      int8   `json:"status"`
}

type ListGroupResult struct {
	List []GroupDTO `json:"list"`
}

func (s *AuthGroupService) listKey() string { return "authgroup:list" }

func (s *AuthGroupService) List(ctx context.Context) (*ListGroupResult, error) {
	if s.Cache != nil {
		if v, _ := s.Cache.Get(ctx, s.listKey()); v != "" {
			if cache.IsNilSentinel(v) {
				metrics.CacheNilHit.Inc()
				return &ListGroupResult{List: []GroupDTO{}}, nil
			}
			var cached ListGroupResult
			if json.Unmarshal([]byte(v), &cached) == nil {
				return &cached, nil
			}
		}
	}
	list, err := s.Groups.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 { // 空结果穿透保护
		if s.Cache != nil {
			_ = s.Cache.SetEX(ctx, s.listKey(), cache.WrapNil(true), cache.JitterTTL(15*time.Second))
		}
		return &ListGroupResult{List: []GroupDTO{}}, nil
	}
	res := make([]GroupDTO, 0, len(list))
	for _, g := range list {
		res = append(res, GroupDTO{ID: g.ID, Name: g.Name, Description: g.Description, Status: g.Status})
	}
	result := &ListGroupResult{List: res}
	if s.Cache != nil {
		b, _ := json.Marshal(result)
		_ = s.Cache.SetEX(ctx, s.listKey(), string(b), cache.JitterTTL(60*time.Second))
	}
	return result, nil
}

type AddGroupParams struct {
	Name, Description string
	Status            int8
}

func (s *AuthGroupService) Add(ctx context.Context, p AddGroupParams) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("name required")
	}
	g := &model.AdminAuthGroup{Name: p.Name, Description: p.Description, Status: p.Status}
	if err := s.Groups.Create(ctx, g); err != nil {
		return err
	}
	s.invalidate()
	return nil
}

type EditGroupParams struct {
	ID                int64
	Name, Description *string
	Status            *int8
}

func (s *AuthGroupService) Edit(ctx context.Context, p EditGroupParams) error {
	if p.ID <= 0 {
		return errors.New("invalid id")
	}
	g, err := s.Groups.FindByID(ctx, p.ID)
	if err != nil {
		return err
	}
	if g == nil {
		return errors.New("not found")
	}
	if p.Name != nil {
		g.Name = *p.Name
	}
	if p.Description != nil {
		g.Description = *p.Description
	}
	if p.Status != nil {
		g.Status = *p.Status
	}
	if err := s.Groups.Update(ctx, g); err != nil {
		return err
	}
	s.invalidate()
	return nil
}

func (s *AuthGroupService) ChangeStatus(ctx context.Context, id int64, status int8) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	if err := s.Groups.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	// 失效该组用户权限
	go s.Perm.InvalidateUsersByGroup(context.Background(), id)
	s.invalidate()
	return nil
}

func (s *AuthGroupService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	if err := s.Groups.Delete(ctx, id); err != nil {
		return err
	}
	go s.Perm.InvalidateUsersByGroup(context.Background(), id)
	s.invalidate()
	return nil
}

func (s *AuthGroupService) DeleteMember(ctx context.Context, gid, uid int64) error {
	if gid <= 0 || uid <= 0 {
		return errors.New("invalid params")
	}
	if err := s.Rel.DeleteMember(ctx, gid, uid); err != nil {
		return err
	}
	// 失效用户权限缓存
	go s.Perm.Invalidate(uid)
	return nil
}

// Touch placeholder for future cache / updated_at logic
func (s *AuthGroupService) Touch(_ context.Context) { _ = time.Now() }

// 缓存失效
func (s *AuthGroupService) invalidate() {
	if s.Cache != nil {
		_ = s.Cache.Del(context.Background(), s.listKey())
	}
}
