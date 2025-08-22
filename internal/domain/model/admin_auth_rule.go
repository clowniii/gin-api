package model

// AdminAuthRule 对应 admin_auth_rule

type AdminAuthRule struct {
	ID      int64  `gorm:"primaryKey" json:"id"`
	URL     string `gorm:"size:80;column:url;index" json:"url"`
	GroupID int64  `gorm:"column:group_id" json:"group_id"`
	Auth    int64  `gorm:"column:auth" json:"auth"`
	Status  int8   `gorm:"column:status" json:"status"`
}

func (AdminAuthRule) TableName() string { return "admin_auth_rule" }
