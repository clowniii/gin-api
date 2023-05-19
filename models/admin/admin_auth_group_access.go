package admin

type AdminAuthGroupAccess struct {
	ID      int64  `db:"id" json:"id"`
	Uid     int64  `db:"uid" json:"uid"`
	GroupId string `db:"group_id" json:"group_id"`

	UpdateData map[string]interface{}
}
