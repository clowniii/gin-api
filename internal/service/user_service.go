package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go-apiadmin/internal/domain/model"
	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/repository/dao"
	"go-apiadmin/pkg/crypto"

	"gorm.io/gorm"
)

type UserService struct {
	Users    *dao.AdminUserDAO
	Groups   *dao.AdminAuthGroupDAO
	GroupRel *dao.AdminAuthGroupAccessDAO
	DB       *gorm.DB
	ListC    cache.Cache // key -> json(ListUsersResult)
	InfoC    cache.Cache // key -> json(UserDTO)
}

func NewUserService(u *dao.AdminUserDAO, g *dao.AdminAuthGroupDAO, gr *dao.AdminAuthGroupAccessDAO, db *gorm.DB) *UserService {
	l1 := cache.NewSimpleAdapter(cache.New(30 * time.Second))
	l1Info := cache.NewSimpleAdapter(cache.New(60 * time.Second))
	return &UserService{Users: u, Groups: g, GroupRel: gr, DB: db, ListC: l1, InfoC: l1Info}
}

// NewUserServiceWithCache 使用统一注入的 cache（例如 LayeredCache），复用同一实例做列表与详情缓存
// 列表 TTL 30s，详情 TTL 60s（调用时分别指定 SetEX 的 ttl）
func NewUserServiceWithCache(u *dao.AdminUserDAO, g *dao.AdminAuthGroupDAO, gr *dao.AdminAuthGroupAccessDAO, db *gorm.DB, c cache.Cache) *UserService {
	return &UserService{Users: u, Groups: g, GroupRel: gr, DB: db, ListC: c, InfoC: c}
}

type ListUsersResult struct {
	List  []UserDTO `json:"list"`
	Total int64     `json:"total"`
}

type UserDTO struct {
	ID         int64             `json:"id"`
	Username   string            `json:"username"`
	Nickname   string            `json:"nickname"`
	Status     int8              `json:"status"`
	CreateTime int64             `json:"create_time"`
	UpdateTime int64             `json:"update_time"`
	CreateIP   int64             `json:"create_ip"`
	Groups     []UserGroupSimple `json:"groups"`
}

type UserGroupSimple struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ListUsersParams struct {
	Username    string
	Status      *int8
	Page, Limit int
}

func (s *UserService) ListUsers(ctx context.Context, p ListUsersParams) (*ListUsersResult, error) {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Limit <= 0 {
		p.Limit = 20
	}
	key := "user:list:" + s.listKey(p)
	if s.ListC != nil {
		if str, _ := s.ListC.Get(ctx, key); str != "" {
			var cached ListUsersResult
			if err := json.Unmarshal([]byte(str), &cached); err == nil {
				return &cached, nil
			}
		}
	}
	res, err := s.listUsersNoCache(ctx, p)
	if err != nil {
		return nil, err
	}
	if s.ListC != nil {
		b, _ := json.Marshal(res)
		_ = s.ListC.SetEX(ctx, key, string(b), 30*time.Second)
	}
	return res, nil
}

func (s *UserService) listUsersNoCache(ctx context.Context, p ListUsersParams) (*ListUsersResult, error) {
	offset := (p.Page - 1) * p.Limit
	users, total, err := s.Users.List(ctx, p.Username, p.Status, offset, p.Limit)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	relMap, _ := s.GroupRel.ListGroupIDsByUsers(ctx, ids)
	uniq := map[int64]struct{}{}
	for _, gids := range relMap {
		for _, gid := range gids {
			uniq[gid] = struct{}{}
		}
	}
	gidList := make([]int64, 0, len(uniq))
	for gid := range uniq {
		gidList = append(gidList, gid)
	}
	groups, _ := s.Groups.FindByIDs(ctx, gidList)
	resSlice := make([]UserDTO, 0, len(users))
	for _, u := range users {
		dto := UserDTO{ID: u.ID, Username: u.Username, Nickname: u.Nickname, Status: u.Status, CreateTime: u.CreateTime, UpdateTime: u.UpdateTime, CreateIP: u.CreateIP}
		if gids, ok := relMap[u.ID]; ok {
			for _, gid := range gids {
				if g, ok2 := groups[gid]; ok2 {
					dto.Groups = append(dto.Groups, UserGroupSimple{ID: g.ID, Name: g.Name})
				}
			}
		}
		resSlice = append(resSlice, dto)
	}
	return &ListUsersResult{List: resSlice, Total: total}, nil
}

type CreateUserParams struct {
	Username, Password, Nickname string
	GroupIDs                     []int64
}

func (s *UserService) CreateUser(ctx context.Context, p CreateUserParams) (int64, error) {
	if p.Username == "" || p.Password == "" {
		return 0, errors.New("missing username/password")
	}
	var newID int64
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now().Unix()
		user := &model.AdminUser{Username: p.Username, Nickname: p.Nickname, Password: crypto.HashPassword(p.Password), CreateTime: now, UpdateTime: now, Status: 1}
		if err := tx.WithContext(ctx).Create(user).Error; err != nil {
			return err
		}
		newID = user.ID
		if err := s.GroupRel.ReplaceUserGroups(ctx, tx, user.ID, p.GroupIDs); err != nil {
			return err
		}
		return nil
	})
	if err == nil {
		s.invalidateAll()
	}
	return newID, err
}

type EditUserParams struct {
	ID       int64
	Nickname string
	Password *string
	Status   *int8
	GroupIDs []int64
}

func (s *UserService) EditUser(ctx context.Context, p EditUserParams) error {
	if p.ID <= 0 {
		return errors.New("invalid id")
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		u, err := s.Users.FindByID(ctx, p.ID)
		if err != nil {
			return err
		}
		if u == nil {
			return errors.New("not found")
		}
		if p.Nickname != "" {
			u.Nickname = p.Nickname
		}
		if p.Status != nil {
			u.Status = *p.Status
		}
		u.UpdateTime = time.Now().Unix()
		if err := s.Users.Update(ctx, u); err != nil {
			return err
		}
		if p.Password != nil && *p.Password != "" {
			if err := s.Users.UpdatePassword(ctx, u.ID, crypto.HashPassword(*p.Password)); err != nil {
				return err
			}
		}
		if err := s.GroupRel.ReplaceUserGroups(ctx, tx, u.ID, p.GroupIDs); err != nil {
			return err
		}
		return nil
	})
	if err == nil {
		s.invalidateUser(p.ID)
	}
	return err
}

func (s *UserService) ChangeStatus(ctx context.Context, id int64, status int8) error {
	err := s.Users.UpdateStatus(ctx, id, status)
	if err == nil {
		s.invalidateUser(id)
	}
	return err
}

func (s *UserService) DeleteUser(ctx context.Context, id int64) error {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.Users.Delete(ctx, id); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where("uid = ?", id).Delete(&model.AdminAuthGroupAccess{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err == nil {
		s.invalidateUser(id)
	}
	return err
}

type UpdateOwnParams struct {
	UID      int64
	Nickname string
	Password *string
}

func (s *UserService) UpdateOwnProfile(ctx context.Context, p UpdateOwnParams) error {
	u, err := s.Users.FindByID(ctx, p.UID)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("not found")
	}
	changed := false
	if p.Nickname != "" {
		u.Nickname = p.Nickname
		changed = true
	}
	if changed {
		u.UpdateTime = time.Now().Unix()
		if err := s.Users.Update(ctx, u); err != nil {
			return err
		}
	}
	if p.Password != nil && *p.Password != "" {
		if err := s.Users.UpdatePassword(ctx, u.ID, crypto.HashPassword(*p.Password)); err != nil {
			return err
		}
	}
	s.invalidateUser(p.UID)
	return nil
}

func (s *UserService) GetUserInfo(ctx context.Context, id int64) (*UserDTO, error) {
	if id <= 0 {
		return nil, errors.New("invalid id")
	}
	key := fmt.Sprint("user:info:", id)
	if s.InfoC != nil {
		if str, _ := s.InfoC.Get(ctx, key); str != "" {
			var dto UserDTO
			if err := json.Unmarshal([]byte(str), &dto); err == nil {
				return &dto, nil
			}
		}
	}
	u, err := s.Users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("not found")
	}
	relMap, _ := s.GroupRel.ListGroupIDsByUsers(ctx, []int64{id})
	var groupsSlice []UserGroupSimple
	if gids, ok := relMap[id]; ok && len(gids) > 0 {
		groups, _ := s.Groups.FindByIDs(ctx, gids)
		for _, gid := range gids {
			if g, ok2 := groups[gid]; ok2 {
				groupsSlice = append(groupsSlice, UserGroupSimple{ID: g.ID, Name: g.Name})
			}
		}
	}
	dto := &UserDTO{ID: u.ID, Username: u.Username, Nickname: u.Nickname, Status: u.Status, CreateTime: u.CreateTime, UpdateTime: u.UpdateTime, CreateIP: u.CreateIP, Groups: groupsSlice}
	if s.InfoC != nil {
		b, _ := json.Marshal(dto)
		_ = s.InfoC.SetEX(ctx, key, string(b), 60*time.Second)
	}
	return dto, nil
}

// ========== 缓存辅助 ==========
func (s *UserService) listKey(p ListUsersParams) string {
	statusVal := int64(-999)
	if p.Status != nil {
		statusVal = int64(*p.Status)
	}
	return p.Username + "|" + fmt.Sprint(statusVal) + "|" + fmt.Sprint(p.Page) + "|" + fmt.Sprint(p.Limit)
}
func (s *UserService) invalidateUser(id int64) {
	if s.InfoC != nil {
		_ = s.InfoC.Del(context.Background(), fmt.Sprint("user:info:", id))
	}
	if s.ListC != nil {
		/* 简化: flush 列表缓存无法精确删除 */
	}
}
func (s *UserService) invalidateAll() {
	if s.InfoC != nil {
		/* optional flush all - 无接口; 依赖实例重建 */
	}
	if s.ListC != nil {
		/* same */
	}
}
