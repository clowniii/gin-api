package model

// AdminField 对应 admin_fields 表
// 增补长度限制与索引

type AdminField struct {
	ID        int64  `gorm:"primaryKey" json:"id"`
	FieldName string `gorm:"column:field_name;size:50" json:"field_name"`
	Hash      string `gorm:"column:hash;size:50;index" json:"hash"`
	DataType  int8   `gorm:"column:data_type" json:"data_type"`
	Default   string `gorm:"column:default;size:500" json:"default"`
	IsMust    int8   `gorm:"column:is_must" json:"is_must"`
	Range     string `gorm:"column:range;size:500" json:"range"`
	Info      string `gorm:"column:info;size:500" json:"info"`
	Type      int8   `gorm:"column:type" json:"type"` // 0=request 1=response
	ShowName  string `gorm:"column:show_name;size:50" json:"show_name"`
}

func (AdminField) TableName() string { return "admin_fields" }
