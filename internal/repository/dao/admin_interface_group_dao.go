package dao

import (
	"context"
	"errors"
	"strings"

	"go-apiadmin/internal/domain/model"

	"gorm.io/gorm"
)

type AdminInterfaceGroupDAO struct{ DB *gorm.DB }

func NewAdminInterfaceGroupDAO(db *gorm.DB) *AdminInterfaceGroupDAO {
	return &AdminInterfaceGroupDAO{DB: db}
}

func (d *AdminInterfaceGroupDAO) FindByID(ctx context.Context, id int64) (*model.AdminInterfaceGroup, error) {
	var m model.AdminInterfaceGroup
	if err := d.DB.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}
func (d *AdminInterfaceGroupDAO) FindByHash(ctx context.Context, hash string) (*model.AdminInterfaceGroup, error) {
	var m model.AdminInterfaceGroup
	if err := d.DB.WithContext(ctx).Where("hash=?", hash).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}
func (d *AdminInterfaceGroupDAO) ExistsName(ctx context.Context, name, appID string, excludeID int64) (bool, error) {
	q := d.DB.WithContext(ctx).Model(&model.AdminInterfaceGroup{}).Where("LOWER(name)=?", strings.ToLower(name))
	if appID != "" {
		q = q.Where("app_id=?", appID)
	}
	if excludeID > 0 {
		q = q.Where("id<>?", excludeID)
	}
	var cnt int64
	if err := q.Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}
func (d *AdminInterfaceGroupDAO) Create(ctx context.Context, m *model.AdminInterfaceGroup) error {
	return d.DB.WithContext(ctx).Create(m).Error
}
func (d *AdminInterfaceGroupDAO) Update(ctx context.Context, m *model.AdminInterfaceGroup) error {
	return d.DB.WithContext(ctx).Model(&model.AdminInterfaceGroup{}).Where("id=?", m.ID).Updates(m).Error
}
func (d *AdminInterfaceGroupDAO) Delete(ctx context.Context, id int64) error {
	return d.DB.WithContext(ctx).Delete(&model.AdminInterfaceGroup{}, id).Error
}
func (d *AdminInterfaceGroupDAO) ChangeStatus(ctx context.Context, id int64, st int8) error {
	return d.DB.WithContext(ctx).Model(&model.AdminInterfaceGroup{}).Where("id=?", id).Update("status", st).Error
}

// List 支持关键词、app_id、status 过滤
func (d *AdminInterfaceGroupDAO) List(ctx context.Context, keywords, appID string, status *int8, page, limit int) ([]model.AdminInterfaceGroup, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	q := d.DB.WithContext(ctx).Model(&model.AdminInterfaceGroup{})
	if keywords != "" {
		q = q.Where("name ILIKE ?", "%"+keywords+"%")
	}
	if appID != "" {
		q = q.Where("app_id=?", appID)
	}
	if status != nil {
		q = q.Where("status=?", *status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AdminInterfaceGroup
	if err := q.Order("sort DESC, id DESC").Limit(limit).Offset((page - 1) * limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
