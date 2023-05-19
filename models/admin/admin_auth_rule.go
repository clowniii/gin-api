package admin

type AdminAuthRule struct {
	ID      int64  `db:"id" json:"id"`
	Url     string `db:"url" json:"url"`           //  规则唯一标识
	GroupId int64  `db:"group_id" json:"group_id"` //  权限所属组的id
	Auth    int64  `db:"auth" json:"auth"`         //  权限数值
	Status  int64  `db:"status" json:"status"`     //  状态：为1正常，为0禁用

	UpdateData map[string]interface{}
}
type AdminAuthRuleSlice []*AdminAuthRule
