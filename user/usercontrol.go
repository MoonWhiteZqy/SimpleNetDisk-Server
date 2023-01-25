package user

import "gorm.io/gorm"

type UserController struct {
	userservice IUserService
}

func (c *UserController) SetSrv(srv IUserService) {
	c.userservice = srv
}

func (c *UserController) Update(u *User, info map[string]string) error {
	return c.userservice.Update(u, info)
}

func (c *UserController) UpdateFriends(u *User, friendid string, db *gorm.DB) error {
	return c.userservice.UpdateFriends(u, friendid, db)
}

func (c *UserController) GetFriends(u *User) ([]string, error) {
	return c.userservice.GetFriends(u)
}
