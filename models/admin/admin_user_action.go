package admin

type AdminUserAction struct {
	ID         int64  `db:"id" json:"id"`
	ActionName string `db:"action_name" json:"action_name"` //  行为名称
	Uid        int64  `db:"uid" json:"uid"`                 //  操作用户id
	Nickname   string `db:"nickname" json:"nickname"`       //  用户昵称
	AddTime    int64  `db:"add_time" json:"add_time"`       //  操作时间
	Data       string `db:"data" json:"data"`               //  用户提交的数据
	Url        string `db:"url" json:"url"`                 //  操作url

	UpdateData map[string]interface{}
}
