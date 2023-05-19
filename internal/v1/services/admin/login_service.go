package admin

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gin-app/internal/v1/services"

	"github.com/gin-gonic/gin"

	"gin-app/models/admin"
	"gin-app/utils"
)

type UserInfo struct {
	ID         int64                `db:"id" json:"id" `
	Username   string               `db:"username" json:"username"`
	Nickname   string               `db:"nickname" json:"nickname"`
	CreateTime int64                `db:"create_time" json:"create_time"`
	CreateIp   int64                `db:"create_ip" json:"create_ip"`
	UpdateTime int64                `db:"update_time" json:"update_time"`
	Status     int64                `db:"status" json:"status"`
	Openid     string               `db:"openid" json:"openid"`
	ApiAuth    string               `json:"apiAuth"`
	Access     []string             `json:"access"`
	Menu       []*admin.AdminMenu   `json:"menu"`
	UserData   *admin.AdminUserData `json:"userData"`
}
type LoginService services.Service
type userChan struct {
	code int
	err  error
}

func (s *LoginService) Login(cu admin.ClientUser, ctx *gin.Context) (interface{}, error) {
	var adminUser = admin.AdminUser{
		UserData: &admin.AdminUserData{UpdateData: make(map[string]interface{})},
		Access:   &admin.AdminAuthGroupAccess{},
		Menu:     []*admin.AdminMenu{},
		AuthRule: []*admin.AdminAuthRule{},
	}

	password := utils.UserMd5(cu.Password)
	err := s.DB.Get(&adminUser, "select * from admin_user where username=? and password=?", cu.Username, password)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("账号或密码错误：%s", err.Error()))
	}
	if adminUser.Password != "" {
		if adminUser.Status == 1 {
			//更新用户数据
			err = adminUser.GetUserData(s.DB)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("获取用户数据失败：%s", err.Error()))
			}
			adminUser.UserData.LoginTimes++
			adminUser.UserData.LastLoginIp = cu.IP
			adminUser.UserData.LastLoginTime = time.Now().Unix()
			adminUser.UserData.UpdateData["login_times"] = adminUser.UserData.LoginTimes
			adminUser.UserData.UpdateData["last_login_ip"] = adminUser.UserData.LastLoginIp
			adminUser.UserData.UpdateData["Last_login_time"] = adminUser.UserData.LastLoginTime
			_, err = adminUser.UserData.CreateOrUpdate(s.DB)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("更新数据失败：%s", err.Error()))
			}
		} else {
			return nil, errors.New("用户已被封禁，请联系管理员")
		}
	} else {
		return nil, errors.New("用户名密码不正确")
	}
	err = adminUser.GetAccess(s.DB)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("获取用户权限数据失败：%s", err.Error()))
	}
	err = adminUser.GetAccessMenuData(s.DB)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("获取当前用户的允许菜单失败：%s", err.Error()))
	}
	apiAuth := md5.Sum([]byte(cu.Password + strconv.Itoa(int(time.Now().Unix()))))
	adminUser.Password = ""
	userInfoData := UserInfo{
		ID:         adminUser.ID,
		Username:   adminUser.Username,
		Nickname:   adminUser.Nickname,
		CreateTime: adminUser.CreateTime,
		CreateIp:   adminUser.CreateIp,
		UpdateTime: adminUser.UpdateTime,
		Status:     adminUser.Status,
		Openid:     adminUser.Openid,
		ApiAuth:    fmt.Sprintf("%x", apiAuth),
		Menu:       utils.GenerateMenuTree(adminUser.Menu, true),
		UserData:   adminUser.UserData,
	}
	for _, v := range adminUser.Menu {
		if v.Url != "" {
			userInfoData.Access = append(userInfoData.Access, v.Url)
		}
	}
	var rdbChan = make(chan userChan)
	go func(u UserInfo, r chan userChan) {
		//检查是否存在
		res, err := s.RDB.HGet(ctx, "user_auth", strconv.Itoa(int(userInfoData.ID))).Result()
		if err == nil {
			err = s.RDB.HDel(ctx, "user_info", res).Err()
			s.RDB.HDel(ctx, "user_expire_time", res)

			if err != nil {
				r <- struct {
					code int
					err  error
				}{code: -1, err: errors.New(fmt.Sprintf("缓存失败0：%s", err.Error()))}
				return
			}
		}

		b, _ := json.Marshal(userInfoData)
		err = s.RDB.HSet(ctx, "user_info", userInfoData.ApiAuth, string(b)).Err()
		if err != nil {
			r <- struct {
				code int
				err  error
			}{code: -1, err: errors.New(fmt.Sprintf("缓存失败1：%s", err.Error()))}
			return

		}
		err = s.RDB.HSet(ctx, "user_auth", userInfoData.ID, userInfoData.ApiAuth).Err()
		if err != nil {
			r <- struct {
				code int
				err  error
			}{code: -1, err: errors.New(fmt.Sprintf("缓存失败2：%s", err.Error()))}
			return
		}
		err = s.RDB.HSet(ctx, "user_expire_time", userInfoData.ApiAuth, time.Now().Format("2006-01-02 15:04:05")).Err()
		if err != nil {
			fmt.Println(err.Error())
		}

		//err = s.RDB.Expire(ctx, fmt.Sprintf("user_info:%s", userInfoData.ApiAuth), time.Second).Err()
		//if err != nil {
		//	fmt.Println(err.Error())
		//}
		//err = s.RDB.Expire(ctx, fmt.Sprintf("user_auth:%d", userInfoData.ID), time.Second).Err()
		//if err != nil {
		//	fmt.Println(err.Error())
		//}

		r <- struct {
			code int
			err  error
		}{code: 0, err: nil}
		return
	}(userInfoData, rdbChan)

	done := <-rdbChan
	if done.err != nil {
		return nil, errors.New(done.err.Error())
	}
	close(rdbChan)

	return userInfoData, nil
}

func (s *LoginService) GetUserInfo(ctx *gin.Context) (interface{}, error) {
	ApiAuth := ctx.GetHeader("Api-Auth")
	//检查是否存在
	res, err := s.RDB.HGet(ctx, "user_info", ApiAuth).Result()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("登录失效，请重新登陆"))
	}
	var userInfo interface{}
	err = json.Unmarshal([]byte(res), &userInfo)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("解码失败：%s", err.Error()))
	}

	return userInfo, nil
}

func (s *LoginService) Logout(ctx *gin.Context) error {
	ApiAuth := ctx.GetHeader("Api-Auth")
	AuthData := s.RDB.HGet(ctx, "user_info", ApiAuth)
	var userInfo admin.AdminUser
	res, err := AuthData.Result()
	if err != nil {
		return errors.New("退出失败，请重试")
	}

	err = json.Unmarshal([]byte(res), &userInfo)
	if err != nil {
		return errors.New("解码失败，请重试")
	}
	err = s.RDB.HDel(ctx, "user_info", userInfo.ApiAuth).Err()
	if err != nil {
		return errors.New(fmt.Sprintf("退出失败0：%s", err.Error()))
	}
	err = s.RDB.HDel(ctx, "user_auth", strconv.Itoa(int(userInfo.ID))).Err()
	if err != nil {
		return errors.New(fmt.Sprintf("退出失败1：%s", err.Error()))
	}
	return nil
}

func (s *LoginService) GetAccessMenu(ctx *gin.Context, id int) (interface{}, error) {
	ApiAuth := ctx.GetHeader("Api-Auth")
	AuthData := s.RDB.HGet(ctx, "user_info", ApiAuth)
	data, err := AuthData.Result()
	if err != nil {
		return nil, errors.New("获取数据失败")
	}
	var userInfo admin.AdminUser
	err = json.Unmarshal([]byte(data), &userInfo)
	if err != nil {
		return nil, errors.New("数据解码失败")
	}
	if id != int(userInfo.ID) {
		return nil, errors.New("用户ID不正确")
	}

	return userInfo.Menu, nil
}
