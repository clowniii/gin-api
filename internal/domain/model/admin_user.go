package model

import "time"

// AdminUser 对应原 admin_user 表
// 密码原为 32 位 MD5，后续可升级 bcrypt
// 采用 GORM 命名，表名通过 TableName 覆盖

type AdminUser struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username   string    `gorm:"size:64;uniqueIndex:uk_username" json:"username"`
	Nickname   string    `gorm:"size:64" json:"nickname"`
	Password   string    `gorm:"size:64" json:"-"` // 预留更长长度
	CreateTime int64     `gorm:"column:create_time" json:"create_time"`
	CreateIP   int64     `gorm:"column:create_ip" json:"create_ip"`
	UpdateTime int64     `gorm:"column:update_time" json:"update_time"`
	Status     int8      `gorm:"column:status" json:"status"`
	OpenID     *string   `gorm:"column:openid;size:100" json:"openid,omitempty"`
	CreatedAt  time.Time `gorm:"->:false;<-:false" json:"-"`
	UpdatedAt  time.Time `gorm:"->:false;<-:false" json:"-"`
}

func (AdminUser) TableName() string { return "admin_user" }
