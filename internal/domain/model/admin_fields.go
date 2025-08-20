package model

// AdminField 对应 admin_fields 表
// 字段规则：用于维护接口请求/响应字段信息

type AdminField struct {
	ID        int64  `gorm:"primaryKey" json:"id"`
	FieldName string `gorm:"column:field_name" json:"field_name"`
	Hash      string `gorm:"column:hash" json:"hash"`
	DataType  int8   `gorm:"column:data_type" json:"data_type"`
	Default   string `gorm:"column:default" json:"default"`
	IsMust    int8   `gorm:"column:is_must" json:"is_must"`
	Range     string `gorm:"column:range" json:"range"`
	Info      string `gorm:"column:info" json:"info"`
	Type      int8   `gorm:"column:type" json:"type"` // 0=request 1=response
	ShowName  string `gorm:"column:show_name" json:"show_name"`
}

func (AdminField) TableName() string { return "admin_fields" }
