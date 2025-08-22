package model

// AdminInterfaceList 对应 admin_list 表 (接口信息)
// 字段含义参考原迁移：api_class, hash, access_token, status, method, info, is_test, return_str, group_hash

type AdminInterfaceList struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	APIClass    string `gorm:"column:api_class;size:50" json:"api_class"`
	Hash        string `gorm:"column:hash;size:50;index" json:"hash"`
	AccessToken int8   `gorm:"column:access_token" json:"access_token"`
	Status      int8   `gorm:"column:status" json:"status"`
	Method      int8   `gorm:"column:method" json:"method"`
	Info        string `gorm:"column:info;size:500" json:"info"`
	IsTest      int8   `gorm:"column:is_test" json:"is_test"`
	ReturnStr   string `gorm:"column:return_str" json:"return_str"`
	GroupHash   string `gorm:"column:group_hash;size:64" json:"group_hash"`
	HashType    int8   `gorm:"column:hash_type" json:"hash_type"` // 1 普通 2 加密
}

func (AdminInterfaceList) TableName() string { return "admin_list" }
