package dao

import (
	"context"
	"go-apiadmin/internal/domain/model"

	"gorm.io/gorm"
)

// AdminAuthGroupAccessDAO handles group access relations.
type AdminAuthGroupAccessDAO struct{ DB *gorm.DB }

func NewAdminAuthGroupAccessDAO(db *gorm.DB) *AdminAuthGroupAccessDAO {
	return &AdminAuthGroupAccessDAO{DB: db}
}

// ListGroupIDsByUser returns group ids for a user.
func (d *AdminAuthGroupAccessDAO) ListGroupIDsByUser(ctx context.Context, uid int64) ([]int64, error) {
	var list []int64
	if err := d.DB.WithContext(ctx).Model(&model.AdminAuthGroupAccess{}).Where("uid = ?", uid).Pluck("group_id", &list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ListGroupIDsByUsers bulk load relations for multiple users.
func (d *AdminAuthGroupAccessDAO) ListGroupIDsByUsers(ctx context.Context, uids []int64) (map[int64][]int64, error) {
	res := make(map[int64][]int64)
	if len(uids) == 0 {
		return res, nil
	}
	var rows []model.AdminAuthGroupAccess
	if err := d.DB.WithContext(ctx).Where("uid IN ?", uids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		res[r.UID] = append(res[r.UID], r.GroupID)
	}
	return res, nil
}

// ListUserIDsByGroup returns user IDs by group id
func (d *AdminAuthGroupAccessDAO) ListUserIDsByGroup(ctx context.Context, gid int64) ([]int64, error) {
	var list []int64
	if err := d.DB.WithContext(ctx).Model(&model.AdminAuthGroupAccess{}).Where("group_id = ?", gid).Pluck("uid", &list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ReplaceUserGroups replace groups of a user (in tx outside).
func (d *AdminAuthGroupAccessDAO) ReplaceUserGroups(ctx context.Context, tx *gorm.DB, uid int64, groupIDs []int64) error {
	if tx == nil {
		tx = d.DB
	}
	if err := tx.WithContext(ctx).Where("uid = ?", uid).Delete(&model.AdminAuthGroupAccess{}).Error; err != nil {
		return err
	}
	if len(groupIDs) == 0 {
		return nil
	}
	rows := make([]model.AdminAuthGroupAccess, 0, len(groupIDs))
	for _, gid := range groupIDs {
		rows = append(rows, model.AdminAuthGroupAccess{UID: uid, GroupID: gid})
	}
	return tx.WithContext(ctx).Create(&rows).Error
}

// DeleteMember 从权限组中移除用户
func (d *AdminAuthGroupAccessDAO) DeleteMember(ctx context.Context, gid, uid int64) error {
	return d.DB.WithContext(ctx).Where("group_id = ? AND uid = ?", gid, uid).Delete(&model.AdminAuthGroupAccess{}).Error
}

// ListMembers 列出组内用户（用于返回或后续扩展）
func (d *AdminAuthGroupAccessDAO) ListMembers(ctx context.Context, gid int64) ([]model.AdminAuthGroupAccess, error) {
	var rows []model.AdminAuthGroupAccess
	if err := d.DB.WithContext(ctx).Where("group_id = ?", gid).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
