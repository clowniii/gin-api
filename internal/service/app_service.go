package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"go-apiadmin/internal/domain/model"
	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/repository/dao"
)

type AppService struct {
	DAO   *dao.AdminAppDAO
	Group *dao.AdminAppGroupDAO
	Cache cache.Cache // 新增: 统一缓存接口（列表+详情）
}

func NewAppService(d *dao.AdminAppDAO, g *dao.AdminAppGroupDAO) *AppService {
	return &AppService{DAO: d, Group: g, Cache: cache.NewSimpleAdapter(cache.New(60 * time.Second))}
}

// NewAppServiceWithCache 允许外部注入 layered cache
func NewAppServiceWithCache(d *dao.AdminAppDAO, g *dao.AdminAppGroupDAO, c cache.Cache) *AppService {
	return &AppService{DAO: d, Group: g, Cache: c}
}

// 随机生成固定长度 token
func randString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:n], nil
}

// generateAppID 生成唯一 app_id
func (s *AppService) generateAppID(ctx context.Context) (string, error) {
	for i := 0; i < 5; i++ {
		id, err := randString(12)
		if err != nil {
			return "", err
		}
		id = strings.ToLower(id)
		// 简单唯一性检查
		m, err := s.DAO.FindByAppID(ctx, id)
		if err != nil {
			return "", err
		}
		if m == nil {
			return id, nil
		}
	}
	return "", errors.New("generate app_id failed")
}

func (s *AppService) generateSecret(ctx context.Context) (string, error) {
	sec, err := randString(24)
	if err != nil {
		return "", err
	}
	return sec, nil
}

type ListAppParams struct {
	Keywords string
	Status   *int8
	Page     int
	Limit    int
}

type AppDTO struct {
	ID        int64  `json:"id"`
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	AppName   string `json:"app_name"`
	AppStatus int8   `json:"app_status"`
	AppInfo   string `json:"app_info"`
	AppGroup  string `json:"app_group"`
}

type ListAppResult struct {
	List  []AppDTO `json:"list"`
	Total int64    `json:"total"`
}

func (s *AppService) List(ctx context.Context, p ListAppParams) (*ListAppResult, error) {
	// 缓存 key
	ck := s.keyList(p)
	if s.Cache != nil {
		if v, _ := s.Cache.Get(ctx, ck); v != "" {
			var cached ListAppResult
			if json.Unmarshal([]byte(v), &cached) == nil {
				return &cached, nil
			}
		}
	}
	list, total, err := s.DAO.List(ctx, p.Keywords, p.Status, p.Page, p.Limit)
	if err != nil {
		return nil, err
	}
	res := make([]AppDTO, 0, len(list))
	for _, m := range list {
		res = append(res, AppDTO{ID: m.ID, AppID: m.AppID, AppSecret: m.AppSecret, AppName: m.AppName, AppStatus: m.AppStatus, AppInfo: m.AppInfo, AppGroup: m.AppGroup})
	}
	result := &ListAppResult{List: res, Total: total}
	if s.Cache != nil {
		b, _ := json.Marshal(result)
		_ = s.Cache.SetEX(ctx, ck, string(b), 60*time.Second)
	}
	return result, nil
}

type AddAppParams struct {
	AppName  string
	AppInfo  string
	AppGroup string
	Status   int8
}

func (s *AppService) Add(ctx context.Context, p AddAppParams) (int64, error) {
	if strings.TrimSpace(p.AppName) == "" {
		return 0, errors.New("app_name required")
	}
	appID, err := s.generateAppID(ctx)
	if err != nil {
		return 0, err
	}
	secret, err := s.generateSecret(ctx)
	if err != nil {
		return 0, err
	}
	m := &model.AdminApp{AppID: appID, AppSecret: secret, AppName: p.AppName, AppStatus: p.Status, AppInfo: p.AppInfo, AppGroup: p.AppGroup, AppAddTime: time.Now().Unix()}
	if err := s.DAO.Create(ctx, m); err != nil {
		return 0, err
	}
	s.invalidateList() // 新增: 失效列表
	return m.ID, nil
}

type EditAppParams struct {
	ID       int64
	AppName  *string
	AppInfo  *string
	AppGroup *string
	Status   *int8
}

func (s *AppService) Edit(ctx context.Context, p EditAppParams) error {
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
	if p.AppName != nil {
		m.AppName = *p.AppName
	}
	if p.AppInfo != nil {
		m.AppInfo = *p.AppInfo
	}
	if p.AppGroup != nil {
		m.AppGroup = *p.AppGroup
	}
	if p.Status != nil {
		m.AppStatus = *p.Status
	}
	if err := s.DAO.Update(ctx, m); err != nil {
		return err
	}
	s.invalidateOne(m.ID)
	return nil
}

func (s *AppService) ChangeStatus(ctx context.Context, id int64, status int8) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	if err := s.DAO.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	s.invalidateOne(id)
	return nil
}
func (s *AppService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	if err := s.DAO.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidateOne(id)
	return nil
}
func (s *AppService) GetInfo(ctx context.Context, id int64) (*AppDTO, error) {
	ck := s.keyInfo(id)
	if s.Cache != nil {
		if v, _ := s.Cache.Get(ctx, ck); v != "" {
			var dto AppDTO
			if json.Unmarshal([]byte(v), &dto) == nil {
				return &dto, nil
			}
		}
	}
	m, err := s.DAO.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errors.New("not found")
	}
	dto := &AppDTO{ID: m.ID, AppID: m.AppID, AppSecret: m.AppSecret, AppName: m.AppName, AppStatus: m.AppStatus, AppInfo: m.AppInfo, AppGroup: m.AppGroup}
	if s.Cache != nil {
		b, _ := json.Marshal(dto)
		_ = s.Cache.SetEX(ctx, ck, string(b), 120*time.Second)
	}
	return dto, nil
}
func (s *AppService) RefreshSecret(ctx context.Context, id int64) (string, error) {
	if id <= 0 {
		return "", errors.New("invalid id")
	}
	sec, err := s.generateSecret(ctx)
	if err != nil {
		return "", err
	}
	if err := s.DAO.UpdateSecret(ctx, id, sec); err != nil {
		return "", err
	}
	s.invalidateOne(id)
	return sec, nil
}

// ========== 缓存辅助 ==========
func (s *AppService) keyList(p ListAppParams) string {
	st := "-"
	if p.Status != nil {
		st = strconv.FormatInt(int64(*p.Status), 10)
	}
	return "app:list:" + p.Keywords + ":" + st + ":" + strconv.Itoa(p.Page) + ":" + strconv.Itoa(p.Limit)
}
func (s *AppService) keyInfo(id int64) string { return "app:info:" + strconv.FormatInt(id, 10) }
func (s *AppService) invalidateOne(id int64) {
	if s.Cache != nil {
		_ = s.Cache.Del(context.Background(), s.keyInfo(id))
	}
	// 列表全部失效困难; 采用 TTL 自然过期 + 可选: 标记版本号
}
func (s *AppService) invalidateList() { /* no-op; rely on TTL */ }
