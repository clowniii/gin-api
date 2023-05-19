package admin

type AdminList struct {
	ID          int64  `db:"id" json:"id"`
	ApiClass    string `db:"api_class" json:"api_class"`       //  api索引，保存了类和方法
	Hash        string `db:"hash" json:"hash"`                 //  api唯一标识
	AccessToken int64  `db:"access_token" json:"access_token"` //  认证方式 1：复杂认证，0：简易认证
	Status      int64  `db:"status" json:"status"`             //  api状态：0表示禁用，1表示启用
	Method      int64  `db:"method" json:"method"`             //  请求方式0：不限1：post，2：get
	Info        string `db:"info" json:"info"`                 //  api中文说明
	IsTest      int64  `db:"is_test" json:"is_test"`           //  是否是测试模式：0:生产模式，1：测试模式
	ReturnStr   string `db:"return_str" json:"return_str"`     //  返回数据示例
	GroupHash   string `db:"group_hash" json:"group_hash"`     //  当前接口所属的接口分组
	HashType    int64  `db:"hash_type" json:"hash_type"`       //  是否采用hash映射， 1：普通模式 2：加密模式

	UpdateData map[string]interface{}
}
