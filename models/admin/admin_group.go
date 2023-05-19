package admin

type AdminGroup struct {
	ID          int64  `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`               //  组名称
	Description string `db:"description" json:"description"` //  组说明
	Status      int64  `db:"status" json:"status"`           //  状态：为1正常，为0禁用
	Hash        string `db:"hash" json:"hash"`               //  组标识
	CreateTime  int64  `db:"create_time" json:"create_time"` //  创建时间
	UpdateTime  int64  `db:"update_time" json:"update_time"` //  修改时间
	Image       string `db:"image" json:"image"`             //  分组封面图
	Hot         int64  `db:"hot" json:"hot"`                 //  分组热度

	UpdateData map[string]interface{}
}
