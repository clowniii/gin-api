package dao

import (
	"context"
	"errors"
	"strings"

	"go-apiadmin/internal/domain/model"

	"gorm.io/gorm"
)

type AdminInterfaceListDAO struct{ DB *gorm.DB }

func NewAdminInterfaceListDAO(db *gorm.DB) *AdminInterfaceListDAO {
	return &AdminInterfaceListDAO{DB: db}
}

func (d *AdminInterfaceListDAO) FindByID(ctx context.Context, id int64) (*model.AdminInterfaceList, error) {
	var m model.AdminInterfaceList
	if err := d.DB.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}
func (d *AdminInterfaceListDAO) FindByHash(ctx context.Context, hash string) (*model.AdminInterfaceList, error) {
	var m model.AdminInterfaceList
	if err := d.DB.WithContext(ctx).Where("hash=?", hash).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}
func (d *AdminInterfaceListDAO) ExistsAPIClass(ctx context.Context, apiClass string, excludeID int64) (bool, error) {
	q := d.DB.WithContext(ctx).Model(&model.AdminInterfaceList{}).Where("LOWER(api_class)=?", strings.ToLower(apiClass))
	if excludeID > 0 {
		q = q.Where("id<>?", excludeID)
	}
	var cnt int64
	if err := q.Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}
func (d *AdminInterfaceListDAO) Create(ctx context.Context, m *model.AdminInterfaceList) error {
	return d.DB.WithContext(ctx).Create(m).Error
}
func (d *AdminInterfaceListDAO) Update(ctx context.Context, m *model.AdminInterfaceList) error {
	return d.DB.WithContext(ctx).Model(&model.AdminInterfaceList{}).Where("id=?", m.ID).Updates(m).Error
}
func (d *AdminInterfaceListDAO) Delete(ctx context.Context, id int64) error {
	return d.DB.WithContext(ctx).Delete(&model.AdminInterfaceList{}, id).Error
}
func (d *AdminInterfaceListDAO) ChangeStatus(ctx context.Context, id int64, st int8) error {
	return d.DB.WithContext(ctx).Model(&model.AdminInterfaceList{}).Where("id=?", id).Update("status", st).Error
}

// List 支持关键词(info)、group_hash、status 过滤
func (d *AdminInterfaceListDAO) List(ctx context.Context, keywords, groupHash string, status *int8, page, limit int) ([]model.AdminInterfaceList, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	q := d.DB.WithContext(ctx).Model(&model.AdminInterfaceList{})
	if keywords != "" {
		q = q.Where("info ILIKE ?", "%"+keywords+"%")
	}
	if groupHash != "" {
		q = q.Where("group_hash=?", groupHash)
	}
	if status != nil {
		q = q.Where("status=?", *status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AdminInterfaceList
	if err := q.Order("id DESC").Limit(limit).Offset((page - 1) * limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (d *AdminInterfaceListDAO) ListAllActive(ctx context.Context) ([]model.AdminInterfaceList, error) {
	var list []model.AdminInterfaceList
	if err := d.DB.WithContext(ctx).Where("status=1").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
