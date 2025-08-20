package dao

import (
	"context"
	"errors"

	"go-apiadmin/internal/domain/model"

	"gorm.io/gorm"
)

type AdminAppGroupDAO struct{ DB *gorm.DB }

func NewAdminAppGroupDAO(db *gorm.DB) *AdminAppGroupDAO { return &AdminAppGroupDAO{DB: db} }

func (d *AdminAppGroupDAO) FindByID(ctx context.Context, id int64) (*model.AdminAppGroup, error) {
	var g model.AdminAppGroup
	if err := d.DB.WithContext(ctx).First(&g, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &g, nil
}
func (d *AdminAppGroupDAO) List(ctx context.Context) ([]model.AdminAppGroup, error) {
	var list []model.AdminAppGroup
	if err := d.DB.WithContext(ctx).Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
func (d *AdminAppGroupDAO) Create(ctx context.Context, g *model.AdminAppGroup) error {
	return d.DB.WithContext(ctx).Create(g).Error
}
func (d *AdminAppGroupDAO) Update(ctx context.Context, g *model.AdminAppGroup) error {
	return d.DB.WithContext(ctx).Model(&model.AdminAppGroup{}).Where("id=?", g.ID).Updates(g).Error
}
func (d *AdminAppGroupDAO) Delete(ctx context.Context, id int64) error {
	return d.DB.WithContext(ctx).Delete(&model.AdminAppGroup{}, id).Error
}
func (d *AdminAppGroupDAO) UpdateStatus(ctx context.Context, id int64, st int8) error {
	return d.DB.WithContext(ctx).Model(&model.AdminAppGroup{}).Where("id=?", id).Update("status", st).Error
}
