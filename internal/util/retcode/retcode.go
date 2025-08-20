package retcode

// Align with legacy ReturnCode.php for compatibility
const (
	SUCCESS              = 1
	INVALID              = -1
	DB_SAVE_ERROR        = -2
	DB_READ_ERROR        = -3
	CACHE_SAVE_ERROR     = -4
	CACHE_READ_ERROR     = -5
	FILE_SAVE_ERROR      = -6
	LOGIN_ERROR          = -7
	NOT_EXISTS           = -8
	JSON_PARSE_FAIL      = -9
	TYPE_ERROR           = -10
	NUMBER_MATCH_ERROR   = -11
	EMPTY_PARAMS         = -12
	DATA_EXISTS          = -13
	AUTH_ERROR           = -14
	OTHER_LOGIN          = -16
	VERSION_INVALID      = -17
	CURL_ERROR           = -18
	RECORD_NOT_FOUND     = -19
	DELETE_FAILED        = -20
	ADD_FAILED           = -21
	UPDATE_FAILED        = -22
	PARAM_INVALID        = -995
	ACCESS_TOKEN_TIMEOUT = -996
	SESSION_TIMEOUT      = -997
	UNKNOWN              = -998
	EXCEPTION            = -999
)

type CodeInfo struct {
	Code    int
	Message string
}

func All() map[string]CodeInfo {
	return map[string]CodeInfo{
		"SUCCESS":              {SUCCESS, "请求成功"},
		"INVALID":              {INVALID, "非法操作"},
		"DB_SAVE_ERROR":        {DB_SAVE_ERROR, "数据存储失败"},
		"DB_READ_ERROR":        {DB_READ_ERROR, "数据读取失败"},
		"CACHE_SAVE_ERROR":     {CACHE_SAVE_ERROR, "缓存存储失败"},
		"CACHE_READ_ERROR":     {CACHE_READ_ERROR, "缓存读取失败"},
		"FILE_SAVE_ERROR":      {FILE_SAVE_ERROR, "文件读取失败"},
		"LOGIN_ERROR":          {LOGIN_ERROR, "登录失败"},
		"NOT_EXISTS":           {NOT_EXISTS, "不存在"},
		"JSON_PARSE_FAIL":      {JSON_PARSE_FAIL, "JSON数据格式错误"},
		"TYPE_ERROR":           {TYPE_ERROR, "类型错误"},
		"NUMBER_MATCH_ERROR":   {NUMBER_MATCH_ERROR, "数字匹配失败"},
		"EMPTY_PARAMS":         {EMPTY_PARAMS, "丢失必要数据"},
		"DATA_EXISTS":          {DATA_EXISTS, "数据已经存在"},
		"AUTH_ERROR":           {AUTH_ERROR, "权限认证失败"},
		"OTHER_LOGIN":          {OTHER_LOGIN, "别的终端登录"},
		"VERSION_INVALID":      {VERSION_INVALID, "API版本非法"},
		"CURL_ERROR":           {CURL_ERROR, "CURL操作异常"},
		"RECORD_NOT_FOUND":     {RECORD_NOT_FOUND, "记录未找到"},
		"DELETE_FAILED":        {DELETE_FAILED, "删除失败"},
		"ADD_FAILED":           {ADD_FAILED, "添加记录失败"},
		"UPDATE_FAILED":        {UPDATE_FAILED, "更新记录失败"},
		"PARAM_INVALID":        {PARAM_INVALID, "数据类型非法"},
		"ACCESS_TOKEN_TIMEOUT": {ACCESS_TOKEN_TIMEOUT, "身份令牌过期"},
		"SESSION_TIMEOUT":      {SESSION_TIMEOUT, "SESSION过期"},
		"UNKNOWN":              {UNKNOWN, "未知错误"},
		"EXCEPTION":            {EXCEPTION, "系统异常"},
	}
}
