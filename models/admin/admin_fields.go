package admin

type AdminFields struct {
	ID        int64  `db:"id" json:"id"`
	FieldName string `db:"field_name" json:"field_name"` //  字段名称
	Hash      string `db:"hash" json:"hash"`             //  权限所属组的id
	DataType  int64  `db:"data_type" json:"data_type"`   //  数据类型，来源于datatype类库
	Default   string `db:"default" json:"default"`       //  默认值
	IsMust    int64  `db:"is_must" json:"is_must"`       //  是否必须 0为不必须，1为必须
	Range     string `db:"range" json:"range"`           //  范围，json字符串，根据数据类型有不一样的含义
	Info      string `db:"info" json:"info"`             //  字段说明
	Type      int64  `db:"type" json:"type"`             //  字段用处：0为request，1为response
	ShowName  string `db:"show_name" json:"show_name"`   //  wiki显示用字段

	UpdateData map[string]interface{}
}
