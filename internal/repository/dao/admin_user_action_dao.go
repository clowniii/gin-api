package dao

import (
	"context"

	"go-apiadmin/internal/domain/model"

	"gorm.io/gorm"
)

type AdminUserActionDAO struct{ DB *gorm.DB }

func NewAdminUserActionDAO(db *gorm.DB) *AdminUserActionDAO { return &AdminUserActionDAO{DB: db} }

func (d *AdminUserActionDAO) List(ctx context.Context, typ int, keywords string, page, limit int) ([]model.AdminUserAction, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	q := d.DB.WithContext(ctx).Model(&model.AdminUserAction{})
	if typ > 0 && keywords != "" {
		switch typ { // 与原 PHP: 1=url,2=nickname,3=uid
		case 1:
			q = q.Where("url ILIKE ?", "%"+keywords+"%")
		case 2:
			q = q.Where("nickname ILIKE ?", "%"+keywords+"%")
		case 3:
			q = q.Where("uid = ?", keywords)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AdminUserAction
	if err := q.Order("add_time DESC").Limit(limit).Offset((page - 1) * limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (d *AdminUserActionDAO) Delete(ctx context.Context, id int64) error {
	return d.DB.WithContext(ctx).Delete(&model.AdminUserAction{}, id).Error
}
