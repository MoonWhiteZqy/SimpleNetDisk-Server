package user

import (
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type IUserService interface {
	// 更新密码
	Update(*User, map[string]string) error
	// 更新好友
	UpdateFriends(*User, string, *gorm.DB) error
	// 获取好友列表
	GetFriends(*User) ([]string, error)
}

type UserServiceImpl struct {
}

// 更新密码或可用磁盘大小
func (srv UserServiceImpl) Update(u *User, info map[string]string) error {
	if u == nil {
		return errors.New("user not exist")
	}

	// 更新密码
	if pwd, ok := info["password"]; ok {
		u.SetPassword(pwd)
	}

	// 更新可用磁盘大小
	if diskstr, ok := info["disk"]; ok {
		disk, err := strconv.ParseInt(diskstr, 10, 64)
		if err != nil {
			return err
		}

		// 防止更新后总空间大小小于已用空间
		if disk < u.GetUseddisk() {
			return errors.New("new disk space too small")
		}
		u.SetDisk(disk)
	}
	return nil
}

// 添加好友,上限为10个
func (srv UserServiceImpl) UpdateFriends(u *User, friendid string, db *gorm.DB) error {
	if friendid == u.Id {
		return errors.New("can't be your own friend")
	}
	friends := u.GetFriends()
	for _, v := range friends {
		if friendid == v {
			return errors.New("friend already exist")
		}
	}
	if len(friends) >= 10 {
		return errors.New("friend limit exceed")
	}
	friends = append(friends, friendid)
	return u.SetFriends(strings.Join(friends, ","), db)
}

// 获取用户的好友列表
func (srv UserServiceImpl) GetFriends(u *User) ([]string, error) {
	res := u.GetFriends()
	return res, nil
}
