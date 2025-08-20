package dao

import (
	"context"
	"go-apiadmin/internal/domain/model"

	"gorm.io/gorm"
)

type AdminAuthRuleDAO struct{ DB *gorm.DB }

func NewAdminAuthRuleDAO(db *gorm.DB) *AdminAuthRuleDAO { return &AdminAuthRuleDAO{DB: db} }

// ListByGroupIDs 根据分组ID批量加载启用的规则
func (d *AdminAuthRuleDAO) ListByGroupIDs(ctx context.Context, gids []int64) ([]model.AdminAuthRule, error) {
	if len(gids) == 0 {
		return []model.AdminAuthRule{}, nil
	}
	var list []model.AdminAuthRule
	if err := d.DB.WithContext(ctx).Where("group_id IN ? AND status = 1", gids).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// FindByID 单条
func (d *AdminAuthRuleDAO) FindByID(ctx context.Context, id int64) (*model.AdminAuthRule, error) {
	var r model.AdminAuthRule
	if err := d.DB.WithContext(ctx).First(&r, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

// ListByIDs 批量
func (d *AdminAuthRuleDAO) ListByIDs(ctx context.Context, ids []int64) ([]model.AdminAuthRule, error) {
	if len(ids) == 0 {
		return []model.AdminAuthRule{}, nil
	}
	var list []model.AdminAuthRule
	if err := d.DB.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (d *AdminAuthRuleDAO) List(ctx context.Context, groupID *int64) ([]model.AdminAuthRule, error) {
	q := d.DB.WithContext(ctx).Model(&model.AdminAuthRule{})
	if groupID != nil {
		q = q.Where("group_id = ?", *groupID)
	}
	var list []model.AdminAuthRule
	if err := q.Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
func (d *AdminAuthRuleDAO) Create(ctx context.Context, r *model.AdminAuthRule) error {
	return d.DB.WithContext(ctx).Create(r).Error
}
func (d *AdminAuthRuleDAO) Update(ctx context.Context, r *model.AdminAuthRule) error {
	return d.DB.WithContext(ctx).Model(&model.AdminAuthRule{}).Where("id=?", r.ID).Updates(r).Error
}
func (d *AdminAuthRuleDAO) Delete(ctx context.Context, id int64) error {
	return d.DB.WithContext(ctx).Delete(&model.AdminAuthRule{}, id).Error
}
func (d *AdminAuthRuleDAO) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return d.DB.WithContext(ctx).Model(&model.AdminAuthRule{}).Where("id=?", id).Update("status", status).Error
}
func (d *AdminAuthRuleDAO) DeleteByGroupID(ctx context.Context, gid int64) error {
	return d.DB.WithContext(ctx).Where("group_id = ?", gid).Delete(&model.AdminAuthRule{}).Error
}
func (d *AdminAuthRuleDAO) ListByGroupID(ctx context.Context, gid int64) ([]model.AdminAuthRule, error) {
	var list []model.AdminAuthRule
	if err := d.DB.WithContext(ctx).Where("group_id = ?", gid).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// DeleteByGroupAndURLs 删除指定组内一批 URL 规则
func (d *AdminAuthRuleDAO) DeleteByGroupAndURLs(ctx context.Context, gid int64, urls []string) error {
	if len(urls) == 0 {
		return nil
	}
	return d.DB.WithContext(ctx).Where("group_id = ? AND url IN ?", gid, urls).Delete(&model.AdminAuthRule{}).Error
}
