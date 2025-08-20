package dao

import (
	"context"
	"errors"
	"go-apiadmin/internal/domain/model"

	"gorm.io/gorm"
)

type AdminFieldsDAO struct{ DB *gorm.DB }

func NewAdminFieldsDAO(db *gorm.DB) *AdminFieldsDAO { return &AdminFieldsDAO{DB: db} }

func (d *AdminFieldsDAO) List(ctx context.Context, hash string, typ int8, page, limit int) ([]model.AdminField, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	q := d.DB.WithContext(ctx).Model(&model.AdminField{}).Where("hash=? AND type=?", hash, typ)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AdminField
	if err := q.Order("id ASC").Limit(limit).Offset((page - 1) * limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (d *AdminFieldsDAO) Create(ctx context.Context, m *model.AdminField) error {
	return d.DB.WithContext(ctx).Create(m).Error
}
func (d *AdminFieldsDAO) Update(ctx context.Context, m *model.AdminField) error {
	return d.DB.WithContext(ctx).Model(&model.AdminField{}).Where("id=?", m.ID).Updates(m).Error
}
func (d *AdminFieldsDAO) Delete(ctx context.Context, id int64) error {
	return d.DB.WithContext(ctx).Delete(&model.AdminField{}, id).Error
}
func (d *AdminFieldsDAO) FindByID(ctx context.Context, id int64) (*model.AdminField, error) {
	var m model.AdminField
	if err := d.DB.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (d *AdminFieldsDAO) DeleteByHash(ctx context.Context, hash string) error {
	return d.DB.WithContext(ctx).Where("hash=?", hash).Delete(&model.AdminField{}).Error
}
func (d *AdminFieldsDAO) ListByHashAndType(ctx context.Context, hash string, typ int8) ([]model.AdminField, error) {
	var list []model.AdminField
	if err := d.DB.WithContext(ctx).Where("hash=? AND type=?", hash, typ).Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
