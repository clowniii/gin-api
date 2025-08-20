package model

// AdminAuthRule 对应 admin_auth_rule
// auth 字段代表权限值(位掩码或整型)，后续结合业务解释

type AdminAuthRule struct {
	ID      int64  `gorm:"primaryKey" json:"id"`
	URL     string `gorm:"size:120;column:url" json:"url"`
	GroupID int64  `gorm:"column:group_id" json:"group_id"`
	Auth    int64  `gorm:"column:auth" json:"auth"`
	Status  int8   `gorm:"column:status" json:"status"`
}

func (AdminAuthRule) TableName() string { return "admin_auth_rule" }
