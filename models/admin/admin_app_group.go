package admin

type AdminAppGroup struct {
	ID          int64  `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`               //  组名称
	Description string `db:"description" json:"description"` //  组说明
	Status      int64  `db:"status" json:"status"`           //  组状态：0表示禁用，1表示启用
	Hash        string `db:"hash" json:"hash"`               //  组标识

	UpdateData map[string]interface{}
}
