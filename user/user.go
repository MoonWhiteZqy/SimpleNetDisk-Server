package user

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Id       string `gorm:"column:user_id"`
	Password string `gorm:"column:password"`
	Friends  string `gorm:"column:friends"`
	Filenum  int    `gorm:"column:file_num"`
	Diskused int64  `gorm:"column:disk_len"`
	Disk     int64  `gorm:"column:disk_cap"`
}

func (u *User) String() string {
	return fmt.Sprintf("id:%v, password:%v, firends:%v, filenum:%v, disk:%v/%v", u.Id, u.Password, u.Friends, u.Filenum, u.Diskused, u.Disk)
}

func GetUser(id string) *User {
	return &User{Id: id}
}

func (u *User) GetId() string {
	return u.Id
}

func (u *User) SetId(id string) bool {
	u.Id = id
	return true
}

func (u *User) GetPassword() string {
	return u.Password
}

func (u *User) SetPassword(pwd string) bool {
	if pwd == u.Password {
		return false
	}
	u.Password = pwd
	return true
}

func (u *User) GetFriends() []string {
	if len(u.Friends) == 0 {
		return make([]string, 0)
	}
	return strings.Split(u.Friends, ",")
}

func (u *User) SetFriends(friends string, db *gorm.DB) error {
	u.Friends = friends
	db.Model(u).Update("friends", u.Friends)
	return db.Error
}

func (u *User) GetFilenum() int {
	return u.Filenum
}

func (u *User) SetFilenum(fnum int) bool {
	if fnum == u.Filenum {
		return false
	}
	u.Filenum = fnum
	return true
}

func (u *User) GetUseddisk() int64 {
	return u.Diskused
}

func (u *User) SetUseddisk(udisk int64) bool {
	if udisk == u.Diskused {
		return false
	}
	u.Diskused = udisk
	return true
}

func (u *User) GetDisk() int64 {
	return u.Disk
}

func (u *User) SetDisk(disk int64) bool {
	u.Disk = disk
	return true
}
