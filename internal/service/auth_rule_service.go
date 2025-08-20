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
)

type AuthRuleService struct {
	Rules *dao.AdminAuthRuleDAO
	Perm  *PermissionService
	Cache cache.Cache // key: authrule:list:<groupID or 0>
}

func NewAuthRuleService(r *dao.AdminAuthRuleDAO, perm *PermissionService) *AuthRuleService {
	return &AuthRuleService{Rules: r, Perm: perm, Cache: cache.NewSimpleAdapter(cache.New(60 * time.Second))}
}

// NewAuthRuleServiceWithCache 外部注入 layered cache
func NewAuthRuleServiceWithCache(r *dao.AdminAuthRuleDAO, perm *PermissionService, c cache.Cache) *AuthRuleService {
	return &AuthRuleService{Rules: r, Perm: perm, Cache: c}
}

type RuleDTO struct {
	ID      int64  `json:"id"`
	URL     string `json:"url"`
	GroupID int64  `json:"group_id"`
	Auth    int64  `json:"auth"`
	Status  int8   `json:"status"`
}

type ListRuleParams struct{ GroupID *int64 }

type ListRuleResult struct {
	List []RuleDTO `json:"list"`
}

func (s *AuthRuleService) listKey(gid *int64) string {
	g := int64(0)
	if gid != nil {
		g = *gid
	}
	return "authrule:list:" + _intToStr(g)
}

func (s *AuthRuleService) List(ctx context.Context, p ListRuleParams) (*ListRuleResult, error) {
	if s.Cache != nil {
		if v, _ := s.Cache.Get(ctx, s.listKey(p.GroupID)); v != "" {
			var cached ListRuleResult
			if json.Unmarshal([]byte(v), &cached) == nil {
				return &cached, nil
			}
		}
	}
	list, err := s.Rules.List(ctx, p.GroupID)
	if err != nil {
		return nil, err
	}
	res := make([]RuleDTO, 0, len(list))
	for _, r := range list {
		res = append(res, RuleDTO{ID: r.ID, URL: r.URL, GroupID: r.GroupID, Auth: r.Auth, Status: r.Status})
	}
	result := &ListRuleResult{List: res}
	if s.Cache != nil {
		b, _ := json.Marshal(result)
		_ = s.Cache.SetEX(ctx, s.listKey(p.GroupID), string(b), 60*time.Second)
	}
	return result, nil
}

type AddRuleParams struct {
	URL     string
	GroupID int64
	Auth    int64
	Status  int8
}

func (s *AuthRuleService) Add(ctx context.Context, p AddRuleParams) error {
	if p.URL == "" {
		return errors.New("url required")
	}
	u := strings.ToLower(p.URL)
	if !strings.HasPrefix(u, "/admin/") {
		u = "/admin/" + strings.TrimPrefix(u, "/")
	}
	obj := &model.AdminAuthRule{URL: u, GroupID: p.GroupID, Auth: p.Auth, Status: p.Status}
	if err := s.Rules.Create(ctx, obj); err != nil {
		return err
	}
	s.invalidate(p.GroupID)
	return nil
}

type EditRuleParams struct {
	ID      int64
	URL     *string
	GroupID *int64
	Auth    *int64
	Status  *int8
}

func (s *AuthRuleService) Edit(ctx context.Context, p EditRuleParams) error {
	if p.ID <= 0 {
		return errors.New("invalid id")
	}
	r, err := s.Rules.FindByID(ctx, p.ID)
	if err != nil {
		return err
	}
	if r == nil {
		return errors.New("not found")
	}
	origGroup := r.GroupID
	if p.URL != nil {
		u := strings.ToLower(*p.URL)
		if !strings.HasPrefix(u, "/admin/") {
			u = "/admin/" + strings.TrimPrefix(u, "/")
		}
		r.URL = u
	}
	if p.GroupID != nil {
		r.GroupID = *p.GroupID
	}
	if p.Auth != nil {
		r.Auth = *p.Auth
	}
	if p.Status != nil {
		r.Status = *p.Status
	}
	if err := s.Rules.Update(ctx, r); err != nil {
		return err
	}
	// 失效权限
	go s.Perm.InvalidateUsersByGroup(context.Background(), r.GroupID)
	s.invalidate(origGroup)
	if r.GroupID != origGroup {
		s.invalidate(r.GroupID)
	}
	return nil
}

func (s *AuthRuleService) ChangeStatus(ctx context.Context, id int64, status int8) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	if err := s.Rules.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	// 简化: 找到该规则所属组，失效其缓存
	r, _ := s.Rules.FindByID(ctx, id)
	if r != nil {
		go s.Perm.InvalidateUsersByGroup(context.Background(), r.GroupID)
		s.invalidate(r.GroupID)
	}
	return nil
}

func (s *AuthRuleService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	r, _ := s.Rules.FindByID(ctx, id)
	if err := s.Rules.Delete(ctx, id); err != nil {
		return err
	}
	if r != nil {
		go s.Perm.InvalidateUsersByGroup(context.Background(), r.GroupID)
		s.invalidate(r.GroupID)
	}
	return nil
}

// BulkEditRules 批量增删规则（用于 /admin/Auth/editRule 兼容）
func (s *AuthRuleService) BulkEditRules(ctx context.Context, groupID int64, rules []string) error {
	if groupID <= 0 {
		return errors.New("invalid group id")
	}
	// 读取现有
	existing, err := s.Rules.ListByGroupID(ctx, groupID)
	if err != nil {
		return err
	}
	existSet := make(map[string]struct{}, len(existing))
	for _, r := range existing {
		existSet[r.URL] = struct{}{}
	}
	// 规范化传入
	addList := make([]*model.AdminAuthRule, 0)
	seen := make(map[string]struct{})
	for _, raw := range rules {
		if raw == "" { // 忽略空
			continue
		}
		u := strings.ToLower(raw)
		if !strings.HasPrefix(u, "/admin/") {
			u = "/admin/" + strings.TrimPrefix(u, "/")
		}
		if _, dup := seen[u]; dup { // 去重
			continue
		}
		seen[u] = struct{}{}
		if _, ok := existSet[u]; !ok { // 需要新增
			addList = append(addList, &model.AdminAuthRule{URL: u, GroupID: groupID, Status: 1})
		} else {
			// 已存在则从 existSet 删除，剩余的是需要删除的
			delete(existSet, u)
		}
	}
	// 批量插入
	for _, r := range addList {
		if err := s.Rules.Create(ctx, r); err != nil {
			return err
		}
	}
	// 剩余 existSet 中的是要删除的
	if len(existSet) > 0 {
		urls := make([]string, 0, len(existSet))
		for u := range existSet {
			urls = append(urls, u)
		}
		if err := s.Rules.DeleteByGroupAndURLs(ctx, groupID, urls); err != nil {
			return err
		}
	}
	// 失效该组用户权限缓存
	go s.Perm.InvalidateUsersByGroup(context.Background(), groupID)
	s.invalidate(groupID)
	return nil
}

// 缓存失效
func (s *AuthRuleService) invalidate(gid int64) {
	if s.Cache != nil {
		_ = s.Cache.Del(context.Background(), s.listKey(&gid))
	}
}
