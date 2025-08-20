package dao

import (
	"context"
	"errors"
	"go-apiadmin/internal/domain/model"

	"gorm.io/gorm"
)

// AdminUserDAO is a data access object for admin users.
type AdminUserDAO struct {
	DB *gorm.DB
}

// NewAdminUserDAO creates a new AdminUserDAO.
func NewAdminUserDAO(db *gorm.DB) *AdminUserDAO { return &AdminUserDAO{DB: db} }

// WithTx returns a DAO bound to the given transaction (or same instance if tx nil).
func (d *AdminUserDAO) WithTx(tx *gorm.DB) *AdminUserDAO {
	if tx == nil {
		return d
	}
	return &AdminUserDAO{DB: tx}
}

// FindByUsername finds an admin user by their username.
func (d *AdminUserDAO) FindByUsername(ctx context.Context, username string) (*model.AdminUser, error) {
	var u model.AdminUser
	if err := d.DB.WithContext(ctx).Where("username = ?", username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// FindByID finds a user by primary id.
func (d *AdminUserDAO) FindByID(ctx context.Context, id int64) (*model.AdminUser, error) {
	var u model.AdminUser
	if err := d.DB.WithContext(ctx).First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// FindByIDs batch fetch users.
func (d *AdminUserDAO) FindByIDs(ctx context.Context, ids []int64) ([]model.AdminUser, error) {
	if len(ids) == 0 {
		return []model.AdminUser{}, nil
	}
	var list []model.AdminUser
	if err := d.DB.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Create inserts a new user record.
func (d *AdminUserDAO) Create(ctx context.Context, u *model.AdminUser) error {
	return d.DB.WithContext(ctx).Create(u).Error
}

// Delete physical delete.
func (d *AdminUserDAO) Delete(ctx context.Context, id int64) error {
	return d.DB.WithContext(ctx).Delete(&model.AdminUser{}, id).Error
}

// Update updates basic editable fields (nickname, status, password(optional handled outside)).
func (d *AdminUserDAO) Update(ctx context.Context, u *model.AdminUser) error {
	return d.DB.WithContext(ctx).Model(&model.AdminUser{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
		"nickname":    u.Nickname,
		"status":      u.Status,
		"update_time": u.UpdateTime,
	}).Error
}

// UpdatePassword updates user's password (expects already hashed / encoded value).
func (d *AdminUserDAO) UpdatePassword(ctx context.Context, id int64, newPwd string) error {
	return d.DB.WithContext(ctx).Model(&model.AdminUser{}).Where("id = ?", id).Update("password", newPwd).Error
}

// UpdateStatus updates user's status.
func (d *AdminUserDAO) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return d.DB.WithContext(ctx).Model(&model.AdminUser{}).Where("id = ?", id).Update("status", status).Error
}

// List returns users with optional filters & pagination. If limit<=0 returns all (capped by default 500).
func (d *AdminUserDAO) List(ctx context.Context, username string, status *int8, offset, limit int) ([]model.AdminUser, int64, error) {
	q := d.DB.WithContext(ctx).Model(&model.AdminUser{})
	if username != "" {
		q = q.Where("username ILIKE ?", "%"+username+"%")
	}
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 500
	}
	var list []model.AdminUser
	if err := q.Offset(offset).Limit(limit).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
