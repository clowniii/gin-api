package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-apiadmin/internal/domain/model"
	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/repository/dao"
)

type LogService struct {
	DAO   *dao.AdminUserActionDAO
	Cache cache.Cache // key -> json(LogListResult)
}

func NewLogService(d *dao.AdminUserActionDAO) *LogService {
	return &LogService{DAO: d, Cache: cache.NewSimpleAdapter(cache.New(30 * time.Second))}
}

type LogListResult struct {
	List  []model.AdminUserAction `json:"list"`
	Count int64                   `json:"count"`
}

func (s *LogService) List(ctx context.Context, typ int, keywords string, page, limit int) (LogListResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	key := s.key(typ, keywords, page, limit)
	if s.Cache != nil {
		if str, _ := s.Cache.Get(ctx, key); str != "" {
			var r LogListResult
			if json.Unmarshal([]byte(str), &r) == nil {
				return r, nil
			}
		}
	}
	list, total, err := s.DAO.List(ctx, typ, keywords, page, limit)
	if err != nil {
		return LogListResult{}, err
	}
	res := LogListResult{List: list, Count: total}
	if s.Cache != nil {
		b, _ := json.Marshal(res)
		_ = s.Cache.SetEX(ctx, key, string(b), 30*time.Second)
	}
	return res, nil
}

func (s *LogService) Delete(ctx context.Context, id int64) error {
	err := s.DAO.Delete(ctx, id)
	if err == nil && s.Cache != nil {
		/* 无法精准; 忽略 */
	}
	return err
}

// ===== cache helpers =====
func (s *LogService) key(typ int, kw string, page, limit int) string {
	return fmt.Sprintf("%d|%s|%d|%d", typ, kw, page, limit)
}
