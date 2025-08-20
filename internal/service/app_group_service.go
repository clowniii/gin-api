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

type AppGroupService struct {
	DAO   *dao.AdminAppGroupDAO
	Cache cache.Cache
}

func NewAppGroupService(d *dao.AdminAppGroupDAO) *AppGroupService {
	return &AppGroupService{DAO: d, Cache: cache.NewSimpleAdapter(cache.New(60 * time.Second))}
}

// NewAppGroupServiceWithCache 外部注入 layered
func NewAppGroupServiceWithCache(d *dao.AdminAppGroupDAO, c cache.Cache) *AppGroupService {
	return &AppGroupService{DAO: d, Cache: c}
}

type AppGroupDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      int8   `json:"status"`
	Hash        string `json:"hash"`
}

type ListAppGroupResult struct {
	List []AppGroupDTO `json:"list"`
}

func (s *AppGroupService) listKey() string { return "appgroup:list" }

func (s *AppGroupService) List(ctx context.Context) (*ListAppGroupResult, error) {
	if s.Cache != nil {
		if v, _ := s.Cache.Get(ctx, s.listKey()); v != "" {
			var cached ListAppGroupResult
			if json.Unmarshal([]byte(v), &cached) == nil {
				return &cached, nil
			}
		}
	}
	list, err := s.DAO.List(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]AppGroupDTO, 0, len(list))
	for _, g := range list {
		res = append(res, AppGroupDTO{ID: g.ID, Name: g.Name, Description: g.Description, Status: g.Status, Hash: g.Hash})
	}
	result := &ListAppGroupResult{List: res}
	if s.Cache != nil {
		b, _ := json.Marshal(result)
		_ = s.Cache.SetEX(ctx, s.listKey(), string(b), 60*time.Second)
	}
	return result, nil
}

type AddAppGroupParams struct {
	Name, Description, Hash string
	Status                  int8
}

type EditAppGroupParams struct {
	ID                      int64
	Name, Description, Hash *string
	Status                  *int8
}

func (s *AppGroupService) Add(ctx context.Context, p AddAppGroupParams) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("name required")
	}
	m := &model.AdminAppGroup{Name: p.Name, Description: p.Description, Status: p.Status, Hash: p.Hash}
	if err := s.DAO.Create(ctx, m); err != nil {
		return err
	}
	s.invalidate()
	return nil
}
func (s *AppGroupService) Edit(ctx context.Context, p EditAppGroupParams) error {
	if p.ID <= 0 {
		return errors.New("invalid id")
	}
	m, err := s.DAO.FindByID(ctx, p.ID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("not found")
	}
	if p.Name != nil {
		m.Name = *p.Name
	}
	if p.Description != nil {
		m.Description = *p.Description
	}
	if p.Hash != nil {
		m.Hash = *p.Hash
	}
	if p.Status != nil {
		m.Status = *p.Status
	}
	if err := s.DAO.Update(ctx, m); err != nil {
		return err
	}
	s.invalidate()
	return nil
}
func (s *AppGroupService) ChangeStatus(ctx context.Context, id int64, status int8) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	err := s.DAO.UpdateStatus(ctx, id, status)
	if err == nil {
		s.invalidate()
	}
	return err
}
func (s *AppGroupService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	err := s.DAO.Delete(ctx, id)
	if err == nil {
		s.invalidate()
	}
	return err
}

func (s *AppGroupService) invalidate() {
	if s.Cache != nil {
		_ = s.Cache.Del(context.Background(), s.listKey())
	}
}

// Touch 兼容
func (s *AppGroupService) Touch(_ context.Context) { _ = time.Now() }
