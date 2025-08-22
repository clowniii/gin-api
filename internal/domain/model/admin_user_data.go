package model

// AdminUserData 对应 admin_user_data 表
// 存储用户登录统计等信息

type AdminUserData struct {
	ID            int64  `gorm:"primaryKey;column:id" json:"id"`
	LoginTimes    int64  `gorm:"column:login_times" json:"login_times"`
	LastLoginIP   int64  `gorm:"column:last_login_ip" json:"last_login_ip"`
	LastLoginTime int64  `gorm:"column:last_login_time" json:"last_login_time"`
	UID           int64  `gorm:"column:uid;index" json:"uid"`
	HeadImg       string `gorm:"column:head_img" json:"head_img"`
}

func (AdminUserData) TableName() string { return "admin_user_data" }
