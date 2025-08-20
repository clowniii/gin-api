package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"go-apiadmin/internal/domain/model"
	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/repository/dao"
)

type InterfaceListService struct {
	DAO   *dao.AdminInterfaceListDAO
	Cache cache.Cache
}

func NewInterfaceListService(d *dao.AdminInterfaceListDAO) *InterfaceListService {
	return &InterfaceListService{DAO: d, Cache: cache.NewSimpleAdapter(cache.New(60 * time.Second))}
}

// NewInterfaceListServiceWithCache 外部注入 layered
func NewInterfaceListServiceWithCache(d *dao.AdminInterfaceListDAO, c cache.Cache) *InterfaceListService {
	return &InterfaceListService{DAO: d, Cache: c}
}

type ListInterfaceParams struct {
	Keywords  string
	GroupHash string
	Status    *int8
	Page      int
	Limit     int
}

type InterfaceDTO struct {
	ID          int64  `json:"id"`
	APIClass    string `json:"api_class"`
	Hash        string `json:"hash"`
	AccessToken int8   `json:"access_token"`
	Status      int8   `json:"status"`
	Method      int8   `json:"method"`
	Info        string `json:"info"`
	IsTest      int8   `json:"is_test"`
	GroupHash   string `json:"group_hash"`
}

type ListInterfaceResult struct {
	List  []InterfaceDTO `json:"list"`
	Total int64          `json:"total"`
}

func (s *InterfaceListService) listKey(p ListInterfaceParams) string {
	st := "-"
	if p.Status != nil {
		st = _intToStr(int64(*p.Status))
	}
	return "iflist:list:" + p.Keywords + ":" + p.GroupHash + ":" + st + ":" + _intToStr(int64(p.Page)) + ":" + _intToStr(int64(p.Limit))
}
func (s *InterfaceListService) infoKeyID(id int64) string      { return "iflist:info:id:" + _intToStr(id) }
func (s *InterfaceListService) infoKeyHash(hash string) string { return "iflist:info:hash:" + hash }

func (s *InterfaceListService) List(ctx context.Context, p ListInterfaceParams) (*ListInterfaceResult, error) {
	if s.Cache != nil {
		if v, _ := s.Cache.Get(ctx, s.listKey(p)); v != "" {
			var cached ListInterfaceResult
			if json.Unmarshal([]byte(v), &cached) == nil {
				return &cached, nil
			}
		}
	}
	list, total, err := s.DAO.List(ctx, p.Keywords, p.GroupHash, p.Status, p.Page, p.Limit)
	if err != nil {
		return nil, err
	}
	res := make([]InterfaceDTO, 0, len(list))
	for _, m := range list {
		res = append(res, InterfaceDTO{ID: m.ID, APIClass: m.APIClass, Hash: m.Hash, AccessToken: m.AccessToken, Status: m.Status, Method: m.Method, Info: m.Info, IsTest: m.IsTest, GroupHash: m.GroupHash})
	}
	result := &ListInterfaceResult{List: res, Total: total}
	if s.Cache != nil {
		b, _ := json.Marshal(result)
		_ = s.Cache.SetEX(ctx, s.listKey(p), string(b), 60*time.Second)
	}
	return result, nil
}

type AddInterfaceParams struct {
	APIClass    string
	AccessToken int8
	Status      int8
	Method      int8
	Info        string
	IsTest      int8
	ReturnStr   string
	GroupHash   string
}

type EditInterfaceParams struct {
	ID          int64
	APIClass    *string
	AccessToken *int8
	Status      *int8
	Method      *int8
	Info        *string
	IsTest      *int8
	ReturnStr   *string
	GroupHash   *string
}

func (s *InterfaceListService) Add(ctx context.Context, p AddInterfaceParams) (int64, error) {
	if strings.TrimSpace(p.APIClass) == "" {
		return 0, errors.New("api_class required")
	}
	if ok, err := s.DAO.ExistsAPIClass(ctx, p.APIClass, 0); err != nil {
		return 0, err
	} else if ok {
		return 0, errors.New("api_class exists")
	}
	// 生成 hash (短 sha1)
	h := sha1.New()
	h.Write([]byte(p.APIClass + time.Now().Format(time.RFC3339Nano)))
	hash := hex.EncodeToString(h.Sum(nil))
	if len(hash) > 32 {
		hash = hash[:32]
	}
	m := &model.AdminInterfaceList{APIClass: p.APIClass, Hash: hash, AccessToken: p.AccessToken, Status: p.Status, Method: p.Method, Info: p.Info, IsTest: p.IsTest, ReturnStr: p.ReturnStr, GroupHash: p.GroupHash}
	if err := s.DAO.Create(ctx, m); err != nil {
		return 0, err
	}
	s.invalidateAll()
	return m.ID, nil
}

func (s *InterfaceListService) Edit(ctx context.Context, p EditInterfaceParams) error {
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
	if p.APIClass != nil {
		m.APIClass = *p.APIClass
	}
	if p.AccessToken != nil {
		m.AccessToken = *p.AccessToken
	}
	if p.Status != nil {
		m.Status = *p.Status
	}
	if p.Method != nil {
		m.Method = *p.Method
	}
	if p.Info != nil {
		m.Info = *p.Info
	}
	if p.IsTest != nil {
		m.IsTest = *p.IsTest
	}
	if p.ReturnStr != nil {
		m.ReturnStr = *p.ReturnStr
	}
	if p.GroupHash != nil {
		m.GroupHash = *p.GroupHash
	}
	if p.APIClass != nil {
		if ok, err := s.DAO.ExistsAPIClass(ctx, m.APIClass, m.ID); err != nil {
			return err
		} else if ok {
			return errors.New("api_class exists")
		}
	}
	if err := s.DAO.Update(ctx, m); err != nil {
		return err
	}
	s.invalidateOne(m.ID, m.Hash)
	return nil
}

func (s *InterfaceListService) ChangeStatus(ctx context.Context, id int64, st int8) error {
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
func (s *InterfaceListService) Delete(ctx context.Context, id int64) error {
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

func (s *InterfaceListService) RefreshRoutes(ctx context.Context, tplPath, outPath string) error {
	// 读取模板
	b, err := ioutil.ReadFile(tplPath)
	if err != nil {
		return err
	}
	list, err := s.DAO.ListAllActive(ctx)
	if err != nil {
		return err
	}
	methodArr := []string{"*", "POST", "GET"}
	lines := make([]string, 0, len(list))
	for _, v := range list {
		mIdx := int(v.Method)
		if mIdx < 0 || mIdx >= len(methodArr) {
			mIdx = 0
		}
		if v.HashType == 1 {
			lines = append(lines, fmt.Sprintf("Route::rule('%s','api.%s','%s')->middleware([app\\\\middleware\\\\ApiAuth::class, app\\\\middleware\\\\ApiPermission::class, app\\\\middleware\\\\RequestFilter::class, app\\\\middleware\\\\ApiLog::class]);", escapePHP(v.APIClass), escapePHP(v.APIClass), methodArr[mIdx]))
		} else {
			lines = append(lines, fmt.Sprintf("Route::rule('%s','api.%s','%s')->middleware([app\\\\middleware\\\\ApiAuth::class, app\\\\middleware\\\\ApiPermission::class, app\\\\middleware\\\\RequestFilter::class, app\\\\middleware\\\\ApiLog::class]);", escapePHP(v.Hash), escapePHP(v.APIClass), methodArr[mIdx]))
		}
	}
	finalStr := strings.Replace(string(b), "{$API_RULE}", strings.Join(lines, "\n    "), 1)
	if err := os.WriteFile(outPath, []byte(finalStr), 0644); err != nil {
		return err
	}
	return nil
}

func escapePHP(s string) string { return strings.ReplaceAll(s, "'", "\\'") }

// ===== 缓存辅助 =====
func (s *InterfaceListService) invalidateOne(id int64, hash string) {
	if s.Cache == nil {
		return
	}
	_ = s.Cache.Del(context.Background(), s.infoKeyID(id), s.infoKeyHash(hash))
}
func (s *InterfaceListService) invalidateAll() { /* rely on TTL */ }
