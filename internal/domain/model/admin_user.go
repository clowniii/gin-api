package model

import "time"

// AdminUser 对应 admin_user 表

type AdminUser struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username   string    `gorm:"size:64;uniqueIndex:uk_username" json:"username"`
	Nickname   string    `gorm:"size:64" json:"nickname"`
	Password   string    `gorm:"size:64" json:"-"` // 旧库 char(32)，保留更长以兼容改进
	CreateTime int64     `gorm:"column:create_time;index" json:"create_time"`
	CreateIP   int64     `gorm:"column:create_ip" json:"create_ip"`
	UpdateTime int64     `gorm:"column:update_time" json:"update_time"`
	Status     int8      `gorm:"column:status" json:"status"`
	OpenID     *string   `gorm:"column:openid;size:100" json:"openid,omitempty"`
	CreatedAt  time.Time `gorm:"->:false;<-:false" json:"-"`
	UpdatedAt  time.Time `gorm:"->:false;<-:false" json:"-"`
}

func (AdminUser) TableName() string { return "admin_user" }
