package handler

import (
	adminh "go-apiadmin/internal/server/http/handler/admin"
	debugh "go-apiadmin/internal/server/http/handler/debug"
	wikih "go-apiadmin/internal/server/http/handler/wiki"
)

// HandlerSet 聚合 admin 与 wiki 子包的 handler，供 router 使用
// 只暴露业务 handler，不再直接暴露依赖。
type HandlerSet struct {
	Auth           *adminh.AuthHandler
	User           *adminh.UserHandler
	Menu           *adminh.MenuHandler
	AuthGroup      *adminh.AuthGroupHandler
	AuthRule       *adminh.AuthRuleHandler
	App            *adminh.AppHandler
	AppGroup       *adminh.AppGroupHandler
	InterfaceGroup *adminh.InterfaceGroupHandler
	InterfaceList  *adminh.InterfaceListHandler
	Fields         *adminh.FieldsHandler
	Log            *adminh.LogHandler
	Cache          *adminh.CacheHandler
	Index          *adminh.IndexHandler
	Wiki           *wikih.WikiHandler
	Debug          *debugh.Handler
}

// NewHandlerSet 创建聚合。参数为子包依赖（各自最小依赖集）。
func NewHandlerSet(ad adminh.Dependencies, wd wikih.Dependencies, dbg debugh.Dependencies) *HandlerSet {
	return &HandlerSet{
		Auth:           adminh.NewAuthHandler(ad),
		User:           adminh.NewUserHandler(ad),
		Menu:           adminh.NewMenuHandler(ad),
		AuthGroup:      adminh.NewAuthGroupHandler(ad),
		AuthRule:       adminh.NewAuthRuleHandler(ad),
		App:            adminh.NewAppHandler(ad),
		AppGroup:       adminh.NewAppGroupHandler(ad),
		InterfaceGroup: adminh.NewInterfaceGroupHandler(ad),
		InterfaceList:  adminh.NewInterfaceListHandler(ad),
		Fields:         adminh.NewFieldsHandler(ad),
		Log:            adminh.NewLogHandler(ad),
		Cache:          adminh.NewCacheHandler(ad),
		Index:          adminh.NewIndexHandler(ad),
		Wiki:           wikih.NewWikiHandler(wd),
		Debug:          debugh.New(dbg),
	}
}
