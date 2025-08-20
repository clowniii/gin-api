package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go-apiadmin/internal/domain/model"
	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/repository/dao"
)

type InterfaceGroupService struct {
	DAO   *dao.AdminInterfaceGroupDAO
	Cache cache.Cache
}

func NewInterfaceGroupService(d *dao.AdminInterfaceGroupDAO) *InterfaceGroupService {
	return &InterfaceGroupService{DAO: d, Cache: cache.NewSimpleAdapter(cache.New(60 * time.Second))}
}

// NewInterfaceGroupServiceWithCache 外部注入 layered
func NewInterfaceGroupServiceWithCache(d *dao.AdminInterfaceGroupDAO, c cache.Cache) *InterfaceGroupService {
	return &InterfaceGroupService{DAO: d, Cache: c}
}

type ListInterfaceGroupParams struct {
	Keywords string
	AppID    string
	Status   *int8
	Page     int
	Limit    int
}

type InterfaceGroupDTO struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	AppID  string `json:"app_id"`
	Status int8   `json:"status"`
	Sort   int    `json:"sort"`
	Remark string `json:"remark"`
	Hash   string `json:"hash"`
}

type ListInterfaceGroupResult struct {
	List  []InterfaceGroupDTO `json:"list"`
	Total int64               `json:"total"`
}

func (s *InterfaceGroupService) listKey(p ListInterfaceGroupParams) string {
	st := "-"
	if p.Status != nil {
		st = _intToStr(int64(*p.Status))
	}
	return "ifgroup:list:" + p.Keywords + ":" + p.AppID + ":" + st + ":" + _intToStr(int64(p.Page)) + ":" + _intToStr(int64(p.Limit))
}
func (s *InterfaceGroupService) infoKeyID(id int64) string      { return "ifgroup:info:id:" + _intToStr(id) }
func (s *InterfaceGroupService) infoKeyHash(hash string) string { return "ifgroup:info:hash:" + hash }

func (s *InterfaceGroupService) List(ctx context.Context, p ListInterfaceGroupParams) (*ListInterfaceGroupResult, error) {
	if s.Cache != nil {
		if v, _ := s.Cache.Get(ctx, s.listKey(p)); v != "" {
			var cached ListInterfaceGroupResult
			if json.Unmarshal([]byte(v), &cached) == nil {
				return &cached, nil
			}
		}
	}
	list, total, err := s.DAO.List(ctx, p.Keywords, p.AppID, p.Status, p.Page, p.Limit)
	if err != nil {
		return nil, err
	}
	res := make([]InterfaceGroupDTO, 0, len(list))
	for _, m := range list {
		res = append(res, InterfaceGroupDTO{ID: m.ID, Name: m.Name, AppID: m.AppID, Status: m.Status, Sort: m.Sort, Remark: m.Remark, Hash: m.Hash})
	}
	result := &ListInterfaceGroupResult{List: res, Total: total}
	if s.Cache != nil {
		b, _ := json.Marshal(result)
		_ = s.Cache.SetEX(ctx, s.listKey(p), string(b), 60*time.Second)
	}
	return result, nil
}

// ===== 原增删改查 =====

type AddInterfaceGroupParams struct {
	Name   string
	AppID  string
	Status int8
	Sort   int
	Remark string
	Hash   string
}

type EditInterfaceGroupParams struct {
	ID     int64
	Name   *string
	AppID  *string
	Status *int8
	Sort   *int
	Remark *string
}

func (s *InterfaceGroupService) Add(ctx context.Context, p AddInterfaceGroupParams) (int64, error) {
	if strings.TrimSpace(p.Name) == "" {
		return 0, errors.New("name required")
	}
	if p.Hash == "" {
		p.Hash = generateShortHash(p.Name + time.Now().Format(time.RFC3339Nano))
	}
	if ok, err := s.DAO.ExistsName(ctx, p.Name, p.AppID, 0); err != nil {
		return 0, err
	} else if ok {
		return 0, errors.New("name exists")
	}
	m := &model.AdminInterfaceGroup{Name: p.Name, AppID: p.AppID, Status: p.Status, Sort: p.Sort, Remark: p.Remark, Hash: p.Hash, AddTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}
	if err := s.DAO.Create(ctx, m); err != nil {
		return 0, err
	}
	s.invalidateAll()
	return m.ID, nil
}

func (s *InterfaceGroupService) Edit(ctx context.Context, p EditInterfaceGroupParams) error {
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
	if p.AppID != nil {
		m.AppID = *p.AppID
	}
	if p.Status != nil {
		m.Status = *p.Status
	}
	if p.Sort != nil {
		m.Sort = *p.Sort
	}
	if p.Remark != nil {
		m.Remark = *p.Remark
	}
	m.UpdateTime = time.Now().Unix()
	if p.Name != nil {
		if ok, err := s.DAO.ExistsName(ctx, m.Name, m.AppID, m.ID); err != nil {
			return err
		} else if ok {
			return errors.New("name exists")
		}
	}
	if err := s.DAO.Update(ctx, m); err != nil {
		return err
	}
	s.invalidateOne(m.ID, m.Hash)
	return nil
}

func (s *InterfaceGroupService) ChangeStatus(ctx context.Context, id int64, st int8) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	err := s.DAO.ChangeStatus(ctx, id, st)
	if err == nil {
		if m, _ := s.DAO.FindByID(ctx, id); m != nil {
			s.invalidateOne(m.ID, m.Hash)
		}
	}
	return err
}
func (s *InterfaceGroupService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	m, _ := s.DAO.FindByID(ctx, id)
	err := s.DAO.Delete(ctx, id)
	if err == nil && m != nil {
		s.invalidateOne(m.ID, m.Hash)
	}
	return err
}

// ===== helper / cache =====
func generateShortHash(src string) string {
	h := sha1.New()
	h.Write([]byte(src))
	s := hex.EncodeToString(h.Sum(nil))
	if len(s) > 16 {
		return s[:16]
	}
	return s
}

func (s *InterfaceGroupService) invalidateOne(id int64, hash string) {
	if s.Cache == nil {
		return
	}
	_ = s.Cache.Del(context.Background(), s.infoKeyID(id), s.infoKeyHash(hash))
	// 列表无法精确: TTL 自然过期
}
func (s *InterfaceGroupService) invalidateAll() { /* rely on TTL */ }
