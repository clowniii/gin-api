package dao

import (
	"context"
	"go-apiadmin/internal/domain/model"

	"gorm.io/gorm"
)

type AdminAuthGroupDAO struct{ DB *gorm.DB }

func NewAdminAuthGroupDAO(db *gorm.DB) *AdminAuthGroupDAO { return &AdminAuthGroupDAO{DB: db} }

func (d *AdminAuthGroupDAO) FindByIDs(ctx context.Context, ids []int64) (map[int64]model.AdminAuthGroup, error) {
	res := make(map[int64]model.AdminAuthGroup)
	if len(ids) == 0 {
		return res, nil
	}
	var list []model.AdminAuthGroup
	if err := d.DB.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for _, g := range list {
		res[g.ID] = g
	}
	return res, nil
}

// FindByID returns single group
func (d *AdminAuthGroupDAO) FindByID(ctx context.Context, id int64) (*model.AdminAuthGroup, error) {
	var g model.AdminAuthGroup
	if err := d.DB.WithContext(ctx).First(&g, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &g, nil
}

func (d *AdminAuthGroupDAO) List(ctx context.Context) ([]model.AdminAuthGroup, error) {
	var list []model.AdminAuthGroup
	if err := d.DB.WithContext(ctx).Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (d *AdminAuthGroupDAO) Create(ctx context.Context, g *model.AdminAuthGroup) error {
	return d.DB.WithContext(ctx).Create(g).Error
}
func (d *AdminAuthGroupDAO) Update(ctx context.Context, g *model.AdminAuthGroup) error {
	return d.DB.WithContext(ctx).Model(&model.AdminAuthGroup{}).Where("id=?", g.ID).Updates(g).Error
}
func (d *AdminAuthGroupDAO) Delete(ctx context.Context, id int64) error {
	return d.DB.WithContext(ctx).Delete(&model.AdminAuthGroup{}, id).Error
}
func (d *AdminAuthGroupDAO) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return d.DB.WithContext(ctx).Model(&model.AdminAuthGroup{}).Where("id=?", id).Update("status", status).Error
}
