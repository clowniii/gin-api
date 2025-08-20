package dao

import (
	"context"
	"go-apiadmin/internal/domain/model"

	"gorm.io/gorm"
)

type AdminGroupDAO struct{ DB *gorm.DB }

func NewAdminGroupDAO(db *gorm.DB) *AdminGroupDAO { return &AdminGroupDAO{DB: db} }

func (d *AdminGroupDAO) ListAll(ctx context.Context) ([]model.AdminGroup, error) {
	var list []model.AdminGroup
	if err := d.DB.WithContext(ctx).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
func (d *AdminGroupDAO) ListActive(ctx context.Context) ([]model.AdminGroup, error) {
	var list []model.AdminGroup
	if err := d.DB.WithContext(ctx).Where("status=1").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
func (d *AdminGroupDAO) FindByHash(ctx context.Context, hash string) (*model.AdminGroup, error) {
	var m model.AdminGroup
	if err := d.DB.WithContext(ctx).Where("hash=?", hash).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}
func (d *AdminGroupDAO) IncrHot(ctx context.Context, hash string) error {
	return d.DB.WithContext(ctx).Model(&model.AdminGroup{}).Where("hash=?", hash).UpdateColumn("hot", gorm.Expr("hot+1")).Error
}
