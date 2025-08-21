package dao

import (
	"context"
	"fmt"
	"go-apiadmin/internal/domain/model"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type AdminAuthRuleDAO struct{ DB *gorm.DB }

func NewAdminAuthRuleDAO(db *gorm.DB) *AdminAuthRuleDAO { return &AdminAuthRuleDAO{DB: db} }

func (d *AdminAuthRuleDAO) tracer() trace.Tracer { return otel.Tracer("dao.admin_auth_rule") }

// ListByGroupIDs 根据分组ID批量加载启用的规则
func (d *AdminAuthRuleDAO) ListByGroupIDs(ctx context.Context, gids []int64) ([]model.AdminAuthRule, error) {
	ctx, span := d.tracer().Start(ctx, "AdminAuthRuleDAO.ListByGroupIDs")
	defer span.End()
	if len(gids) == 0 {
		return []model.AdminAuthRule{}, nil
	}
	var list []model.AdminAuthRule
	if err := d.DB.WithContext(ctx).Where("group_id IN ? AND status = 1", gids).Find(&list).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("list rules by groups: %w", err)
	}
	return list, nil
}

// FindByID 单条
func (d *AdminAuthRuleDAO) FindByID(ctx context.Context, id int64) (*model.AdminAuthRule, error) {
	ctx, span := d.tracer().Start(ctx, "AdminAuthRuleDAO.FindByID")
	defer span.End()
	var r model.AdminAuthRule
	if err := d.DB.WithContext(ctx).First(&r, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("find rule id=%d: %w", id, err)
	}
	return &r, nil
}

// ListByIDs 批量
func (d *AdminAuthRuleDAO) ListByIDs(ctx context.Context, ids []int64) ([]model.AdminAuthRule, error) {
	ctx, span := d.tracer().Start(ctx, "AdminAuthRuleDAO.ListByIDs")
	defer span.End()
	if len(ids) == 0 {
		return []model.AdminAuthRule{}, nil
	}
	var list []model.AdminAuthRule
	if err := d.DB.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("list rules by ids: %w", err)
	}
	return list, nil
}

func (d *AdminAuthRuleDAO) List(ctx context.Context, groupID *int64) ([]model.AdminAuthRule, error) {
	ctx, span := d.tracer().Start(ctx, "AdminAuthRuleDAO.List")
	defer span.End()
	q := d.DB.WithContext(ctx).Model(&model.AdminAuthRule{})
	if groupID != nil {
		q = q.Where("group_id = ?", *groupID)
	}
	var list []model.AdminAuthRule
	if err := q.Order("id DESC").Find(&list).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("list rules: %w", err)
	}
	return list, nil
}
func (d *AdminAuthRuleDAO) Create(ctx context.Context, r *model.AdminAuthRule) error {
	ctx, span := d.tracer().Start(ctx, "AdminAuthRuleDAO.Create")
	defer span.End()
	if err := d.DB.WithContext(ctx).Create(r).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("create rule: %w", err)
	}
	return nil
}
func (d *AdminAuthRuleDAO) Update(ctx context.Context, r *model.AdminAuthRule) error {
	ctx, span := d.tracer().Start(ctx, "AdminAuthRuleDAO.Update")
	defer span.End()
	if err := d.DB.WithContext(ctx).Model(&model.AdminAuthRule{}).Where("id=?", r.ID).Updates(r).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("update rule id=%d: %w", r.ID, err)
	}
	return nil
}
func (d *AdminAuthRuleDAO) Delete(ctx context.Context, id int64) error {
	ctx, span := d.tracer().Start(ctx, "AdminAuthRuleDAO.Delete")
	defer span.End()
	if err := d.DB.WithContext(ctx).Delete(&model.AdminAuthRule{}, id).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("delete rule id=%d: %w", id, err)
	}
	return nil
}
func (d *AdminAuthRuleDAO) UpdateStatus(ctx context.Context, id int64, status int8) error {
	ctx, span := d.tracer().Start(ctx, "AdminAuthRuleDAO.UpdateStatus")
	defer span.End()
	if err := d.DB.WithContext(ctx).Model(&model.AdminAuthRule{}).Where("id=?", id).Update("status", status).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("update status id=%d: %w", id, err)
	}
	return nil
}
func (d *AdminAuthRuleDAO) DeleteByGroupID(ctx context.Context, gid int64) error {
	ctx, span := d.tracer().Start(ctx, "AdminAuthRuleDAO.DeleteByGroupID")
	defer span.End()
	if err := d.DB.WithContext(ctx).Where("group_id = ?", gid).Delete(&model.AdminAuthRule{}).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("delete by group id=%d: %w", gid, err)
	}
	return nil
}
func (d *AdminAuthRuleDAO) ListByGroupID(ctx context.Context, gid int64) ([]model.AdminAuthRule, error) {
	ctx, span := d.tracer().Start(ctx, "AdminAuthRuleDAO.ListByGroupID")
	defer span.End()
	var list []model.AdminAuthRule
	if err := d.DB.WithContext(ctx).Where("group_id = ?", gid).Find(&list).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("list by group id=%d: %w", gid, err)
	}
	return list, nil
}

// DeleteByGroupAndURLs 删除指定组内一批 URL 规则
func (d *AdminAuthRuleDAO) DeleteByGroupAndURLs(ctx context.Context, gid int64, urls []string) error {
	ctx, span := d.tracer().Start(ctx, "AdminAuthRuleDAO.DeleteByGroupAndURLs")
	defer span.End()
	if len(urls) == 0 {
		return nil
	}
	if err := d.DB.WithContext(ctx).Where("group_id = ? AND url IN ?", gid, urls).Delete(&model.AdminAuthRule{}).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("delete by group&urls gid=%d: %w", gid, err)
	}
	return nil
}
