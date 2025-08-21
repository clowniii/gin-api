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

// AdminAuthGroupAccessDAO handles group access relations.
type AdminAuthGroupAccessDAO struct{ DB *gorm.DB }

func NewAdminAuthGroupAccessDAO(db *gorm.DB) *AdminAuthGroupAccessDAO {
	return &AdminAuthGroupAccessDAO{DB: db}
}

func (d *AdminAuthGroupAccessDAO) tracer() trace.Tracer {
	return otel.Tracer("dao.admin_auth_group_access")
}

// ListGroupIDsByUser returns group ids for a user.
func (d *AdminAuthGroupAccessDAO) ListGroupIDsByUser(ctx context.Context, uid int64) ([]int64, error) {
	ctx, span := d.tracer().Start(ctx, "AdminAuthGroupAccessDAO.ListGroupIDsByUser")
	defer span.End()
	var list []int64
	if err := d.DB.WithContext(ctx).Model(&model.AdminAuthGroupAccess{}).Where("uid = ?", uid).Pluck("group_id", &list).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("list group ids by user uid=%d: %w", uid, err)
	}
	return list, nil
}

// ListGroupIDsByUsers bulk load relations for multiple users.
func (d *AdminAuthGroupAccessDAO) ListGroupIDsByUsers(ctx context.Context, uids []int64) (map[int64][]int64, error) {
	ctx, span := d.tracer().Start(ctx, "AdminAuthGroupAccessDAO.ListGroupIDsByUsers")
	defer span.End()
	res := make(map[int64][]int64)
	if len(uids) == 0 {
		return res, nil
	}
	var rows []model.AdminAuthGroupAccess
	if err := d.DB.WithContext(ctx).Where("uid IN ?", uids).Find(&rows).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("list group ids by users: %w", err)
	}
	for _, r := range rows {
		res[r.UID] = append(res[r.UID], r.GroupID)
	}
	return res, nil
}

// ListUserIDsByGroup returns user IDs by group id
func (d *AdminAuthGroupAccessDAO) ListUserIDsByGroup(ctx context.Context, gid int64) ([]int64, error) {
	ctx, span := d.tracer().Start(ctx, "AdminAuthGroupAccessDAO.ListUserIDsByGroup")
	defer span.End()
	var list []int64
	if err := d.DB.WithContext(ctx).Model(&model.AdminAuthGroupAccess{}).Where("group_id = ?", gid).Pluck("uid", &list).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("list user ids by group gid=%d: %w", gid, err)
	}
	return list, nil
}

// ReplaceUserGroups replace groups of a user (in tx outside).
func (d *AdminAuthGroupAccessDAO) ReplaceUserGroups(ctx context.Context, tx *gorm.DB, uid int64, groupIDs []int64) error {
	ctx, span := d.tracer().Start(ctx, "AdminAuthGroupAccessDAO.ReplaceUserGroups")
	defer span.End()
	if tx == nil {
		tx = d.DB
	}
	if err := tx.WithContext(ctx).Where("uid = ?", uid).Delete(&model.AdminAuthGroupAccess{}).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("replace user groups (delete) uid=%d: %w", uid, err)
	}
	if len(groupIDs) == 0 {
		return nil
	}
	rows := make([]model.AdminAuthGroupAccess, 0, len(groupIDs))
	for _, gid := range groupIDs {
		rows = append(rows, model.AdminAuthGroupAccess{UID: uid, GroupID: gid})
	}
	if err := tx.WithContext(ctx).Create(&rows).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("replace user groups (insert) uid=%d: %w", uid, err)
	}
	return nil
}

// DeleteMember 从权限组中移除用户
func (d *AdminAuthGroupAccessDAO) DeleteMember(ctx context.Context, gid, uid int64) error {
	ctx, span := d.tracer().Start(ctx, "AdminAuthGroupAccessDAO.DeleteMember")
	defer span.End()
	if err := d.DB.WithContext(ctx).Where("group_id = ? AND uid = ?", gid, uid).Delete(&model.AdminAuthGroupAccess{}).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("delete member gid=%d uid=%d: %w", gid, uid, err)
	}
	return nil
}

// ListMembers 列出组内用户（用于返回或后续扩展）
func (d *AdminAuthGroupAccessDAO) ListMembers(ctx context.Context, gid int64) ([]model.AdminAuthGroupAccess, error) {
	ctx, span := d.tracer().Start(ctx, "AdminAuthGroupAccessDAO.ListMembers")
	defer span.End()
	var rows []model.AdminAuthGroupAccess
	if err := d.DB.WithContext(ctx).Where("group_id = ?", gid).Find(&rows).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("list members gid=%d: %w", gid, err)
	}
	return rows, nil
}
