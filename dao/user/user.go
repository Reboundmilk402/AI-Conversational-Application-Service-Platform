package user

import (
	"GopherAI/common/mysql"
	"GopherAI/model"
	"GopherAI/utils"

	"gorm.io/gorm"
)

const (
	CodeMsg     = "GopherAI验证码如下(验证码仅限于2分钟有效): "
	UserNameMsg = "GopherAI的账号如下，请保留好，后续可以用账号进行登录 "
)

func IsExistUser(username string) (bool, *model.User) {
	user, err := mysql.GetUserByUsername(username)
	if err == gorm.ErrRecordNotFound || user == nil {
		return false, nil
	}
	return true, user
}

func IsExistEmail(email string) (bool, *model.User) {
	user, err := mysql.GetUserByEmail(email)
	if err == gorm.ErrRecordNotFound || user == nil {
		return false, nil
	}
	return true, user
}

func IsExistLoginAccount(account string) (bool, *model.User) {
	if ok, user := IsExistUser(account); ok {
		return true, user
	}
	if ok, user := IsExistEmail(account); ok {
		return true, user
	}
	return false, nil
}

func Register(username, email, password string) (*model.User, bool) {
	user, err := mysql.InsertUser(&model.User{
		Email:    email,
		Name:     username,
		Username: username,
		Password: utils.MD5(password),
	})
	if err != nil {
		return nil, false
	}
	return user, true
}
