package dao

import (
	"context"
	"errors"
	"strings"

	"go-apiadmin/internal/domain/model"

	"gorm.io/gorm"
)

type AdminAppDAO struct{ DB *gorm.DB }

func NewAdminAppDAO(db *gorm.DB) *AdminAppDAO { return &AdminAppDAO{DB: db} }

func (d *AdminAppDAO) FindByID(ctx context.Context, id int64) (*model.AdminApp, error) {
	var m model.AdminApp
	if err := d.DB.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}
func (d *AdminAppDAO) FindByAppID(ctx context.Context, appID string) (*model.AdminApp, error) {
	var m model.AdminApp
	if err := d.DB.WithContext(ctx).Where("app_id = ?", appID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}
func (d *AdminAppDAO) FindByAppIDAndSecret(ctx context.Context, appID, secret string) (*model.AdminApp, error) {
	var m model.AdminApp
	if err := d.DB.WithContext(ctx).Where("app_id=? AND app_secret=?", appID, secret).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// List 支持关键词与状态过滤，分页
func (d *AdminAppDAO) List(ctx context.Context, keywords string, status *int8, page, limit int) ([]model.AdminApp, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	q := d.DB.WithContext(ctx).Model(&model.AdminApp{})
	if keywords != "" {
		q = q.Where("app_name ILIKE ?", "%"+keywords+"%")
	}
	if status != nil {
		q = q.Where("app_status = ?", *status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AdminApp
	if err := q.Order("id DESC").Limit(limit).Offset((page - 1) * limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (d *AdminAppDAO) Create(ctx context.Context, m *model.AdminApp) error {
	return d.DB.WithContext(ctx).Create(m).Error
}
func (d *AdminAppDAO) Update(ctx context.Context, m *model.AdminApp) error {
	return d.DB.WithContext(ctx).Model(&model.AdminApp{}).Where("id=?", m.ID).Updates(m).Error
}
func (d *AdminAppDAO) Delete(ctx context.Context, id int64) error {
	return d.DB.WithContext(ctx).Delete(&model.AdminApp{}, id).Error
}
func (d *AdminAppDAO) UpdateStatus(ctx context.Context, id int64, st int8) error {
	return d.DB.WithContext(ctx).Model(&model.AdminApp{}).Where("id=?", id).Update("app_status", st).Error
}
func (d *AdminAppDAO) UpdateSecret(ctx context.Context, id int64, secret string) error {
	return d.DB.WithContext(ctx).Model(&model.AdminApp{}).Where("id=?", id).Update("app_secret", secret).Error
}

// BulkByIDs 批量载入
func (d *AdminAppDAO) BulkByIDs(ctx context.Context, ids []int64) (map[int64]model.AdminApp, error) {
	res := make(map[int64]model.AdminApp)
	if len(ids) == 0 {
		return res, nil
	}
	var list []model.AdminApp
	if err := d.DB.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for _, m := range list {
		res[m.ID] = m
	}
	return res, nil
}

// SearchAppIDs 用于校验 app_id 是否冲突
func (d *AdminAppDAO) SearchAppIDs(ctx context.Context, prefix string) ([]string, error) {
	var list []string
	if err := d.DB.WithContext(ctx).Model(&model.AdminApp{}).Where("app_id ILIKE ?", strings.ToLower(prefix)+"%").Pluck("app_id", &list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
