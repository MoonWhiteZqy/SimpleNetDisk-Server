package main

import (
	"encoding/json"
	"file"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"user"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type RawUser struct {
	UserID   string `json:"user_id"`
	Password string `json:"password"`
	Disk     int64  `json:"disk"`
}

type SimpleFile struct {
	FilePath string   `json:"file_path"`
	Uploader string   `json:"uploader"`
	Target   []string `json:"target"`
}

type FriendMsg struct {
	Me     string `json:"me"`
	Friend string `json:"friend"`
}

type DeleteMsg struct {
	UserID    string `json:"user_id"`
	ManagerID string `json:"manager_id"`
}

type TargetMsg struct {
	UserID string `json:"user_id"`
	Target string `json:"target"`
	Path   string `json:"path"`
}

type QueryMsg struct {
	UserID        string `json:"user_id"`
	LowerSpace    string `json:"lower_space"`
	HigherSpace   string `json:"higher_space"`
	LowerFileNum  string `json:"lower_file_num"`
	HigherFileNum string `json:"higher_file_num"`
}

// 数据库全局对象
var db *gorm.DB

// 用户列表
var userMap map[string]*user.User
var userLock sync.Mutex

// 用户可下载的文件列表
var fileOwnerMap map[string][]*file.File
var fileLock sync.Mutex

// 文件列表
var fileMap map[string]*file.File

// 返回失败HTTP响应
//
// Json{"status", "reason"}
func fail(ctx *gin.Context, reason string) {
	ctx.JSON(http.StatusOK, gin.H{
		"status": "fail",
		"reason": reason,
	})
}

func init() {
	var err error
	var userlist []user.User
	var filelist []file.File
	db, err = gorm.Open(mysql.Open("gorm:gorm@tcp(127.0.0.1:9910)/gorm?parseTime=true"))
	if err != nil {
		log.Fatalf("%v when init db", err.Error())
	}
	db.AutoMigrate(&user.User{}, &file.File{})

	// 获取数据库内用户
	db.Find(&userlist)
	if db.Error != nil {
		log.Fatalf("%v when init user slice", db.Error)
	}
	userMap = make(map[string]*user.User, len(userlist))
	for i := range userlist {
		userMap[userlist[i].Id] = &userlist[i]
	}

	// 获取数据库内文件信息
	fileOwnerMap = make(map[string][]*file.File, len(userlist))
	db.Find(&filelist)
	if db.Error != nil {
		log.Fatalf("%v when init file slice", db.Error)
	}
	fileMap = make(map[string]*file.File, len(filelist))
	for i := range filelist {
		fileMap[filelist[i].GetPath()] = &filelist[i]
		owner, target := filelist[i].GetUploader(), filelist[i].GetTarget()
		fileOwnerMap[owner] = append(fileOwnerMap[owner], &filelist[i])
		for _, u := range target {
			fileOwnerMap[u] = append(fileOwnerMap[u], &filelist[i])
		}
	}
}

func main() {
	r := gin.Default()
	ug := r.Group("user")
	{
		ug.POST("register", UserRegisterHandler())
		ug.POST("login", UserLoginHandler())
		ug.GET("list", UserListHandler())
		ug.GET("files/:user_id", UserFilesHandler())
		ug.GET("friends/:user_id", UserFriendsListHandler())
		ug.POST("update/friend", UserAddFriendHandler())
	}
	mg := r.Group("manager")
	{
		mg.POST("delete", ManagerDeleteHandler())
		mg.POST("query", ManagerQueryHandler())
	}
	fg := r.Group("file")
	{
		fg.POST("upload/:user/:path", FileUploadHandler())
		fg.POST("target", FileTargetHandler())
		fg.GET("owner", FileOwnerHandler())
		fg.POST("download", FileDownloadHandler())
		fg.POST("delete", FileDeleteHandler())
	}
	r.Run("127.0.0.1:8080")
}

// 用户注册
//
// 输入:Json{"user_id", "password", "disk"}
//
// 若成功,创建新用户
//
// 返回:Json{"status", "user_id"/"reason"}
func UserRegisterHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userLock.Lock()
		defer userLock.Unlock()
		var u RawUser
		body, _ := ioutil.ReadAll(ctx.Request.Body)
		json.Unmarshal(body, &u)

		// 判断是否存在同名用户，不允许重复注册
		if _, ok := userMap[u.UserID]; ok {
			fail(ctx, "user already exist")
			return
		}

		// 生成用户数据
		current_user := user.User{Id: u.UserID, Password: u.Password, Disk: u.Disk}
		db.Create(&current_user)
		if db.Error != nil {
			log.Printf("%v when create user", db.Error)
		}

		// 加入数据结构中
		userMap[u.UserID] = &current_user

		// 提示注册成功
		ctx.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"user_id": u.UserID,
		})
	}
}

// 用户登录
//
// 输入:Json{"user_id", "password"}
//
// 返回:Json{"status", "reason"/"user_id"}
func UserLoginHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var msg RawUser
		body, _ := ioutil.ReadAll(ctx.Request.Body)
		json.Unmarshal(body, &msg)

		userLock.Lock()
		defer userLock.Unlock()
		u := userMap[msg.UserID]
		if u == nil || u.GetPassword() != msg.Password {
			fail(ctx, "user not exist")
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"user_id": msg.UserID,
		})
	}
}

// 获取已注册用户信息,调试用
//
// 返回Json{"<user_id>":user.User...}
func UserListHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userLock.Lock()
		defer userLock.Unlock()
		ctx.JSON(http.StatusOK, userMap)
	}
}

// 获取用户可以下载的文件列表
//
// 返回:Json{"my_file", "other_file", "file_num", "space_used"}
func UserFilesHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fileLock.Lock()
		defer fileLock.Unlock()
		uid := ctx.Param("user_id")
		u := userMap[uid]
		if u == nil {
			fail(ctx, "user not exist")
			return
		}

		// 自己上传的文件列表
		myFile := make([]SimpleFile, 0)
		// 由别人分享的文件列表
		otherFile := make([]SimpleFile, 0)
		for _, f := range fileOwnerMap[uid] {
			if uid == f.GetUploader() {
				myFile = append(myFile, SimpleFile{FilePath: f.GetPath(), Uploader: uid, Target: f.GetTarget()})
			} else {
				otherFile = append(otherFile, SimpleFile{FilePath: f.GetPath(), Uploader: f.GetUploader(), Target: make([]string, 0)})
			}
		}
		ctx.JSON(http.StatusOK, gin.H{
			"my_file":    myFile,
			"other_file": otherFile,
			"file_num":   u.GetFilenum(),
			"space_used": fmt.Sprintf("%v/%v", u.GetUseddisk(), u.GetDisk()),
		})
	}
}

// 用户添加好友
//
// 输入:Json{"me", "friend"}
//
// 返回:Json{"status", "reason"}
func UserAddFriendHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fileLock.Lock()
		defer fileLock.Unlock()
		var m FriendMsg
		body, _ := ioutil.ReadAll(ctx.Request.Body)
		json.Unmarshal(body, &m)

		if _, ok := userMap[m.Me]; !ok {
			fail(ctx, "me not exist")
			return
		}

		if _, ok := userMap[m.Friend]; !ok {
			fail(ctx, "target not exist")
			return
		}

		ctl := &user.UserController{}
		ctl.SetSrv(user.UserServiceImpl{})
		err := ctl.UpdateFriends(userMap[m.Me], m.Friend, db)
		if err != nil {
			fail(ctx, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"status": "success",
		})
	}
}

// 管理员删除用户
//
// 输入:Json{"manager_id", "user_id"}
//
// 返回:Json{"status", "reason"}
func ManagerDeleteHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userLock.Lock()
		defer userLock.Unlock()
		var msg DeleteMsg
		// 读取要删除的用户名
		body, _ := ioutil.ReadAll(ctx.Request.Body)
		json.Unmarshal(body, &msg)

		for _, u := range strings.Split(msg.UserID, ",") {
			// 删除用户
			if _, ok := userMap[u]; ok {
				db.Delete(userMap[u])
			}
			if db.Error != nil {
				fail(ctx, db.Error.Error())
				return
			}
			// 将用户从内存用户表中删除
			delete(userMap, u)
		}
		ctx.JSON(http.StatusOK, gin.H{
			"status": "success",
		})
	}
}

// 管理员查找用户
//
// 输入:Json{"user_id", "lower_space", "higher_space", "lower_file_num", "higher_file_num"}
//
// 输出:Json{"users":[]User}
func ManagerQueryHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var msg QueryMsg
		body, _ := ioutil.ReadAll(ctx.Request.Body)
		json.Unmarshal(body, &msg)

		var users []user.User

		if len(msg.UserID) > 0 {
			db.Where("user_id = ?", msg.UserID).Find(&users)
		} else {
			db.Where("disk_len > ?", msg.LowerSpace).Where("disk_len < ?", msg.HigherSpace).Where("file_num > ?", msg.LowerFileNum).Where("file_num < ?", msg.HigherFileNum).Find(&users)
		}
		ctx.JSON(http.StatusOK, gin.H{
			"users": users,
		})
	}
}

// 获取用户的好友
//
// 返回:Json{"status", "friends"}
func UserFriendsListHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fileLock.Lock()
		defer fileLock.Unlock()
		uid := ctx.Param("user_id")

		if _, ok := userMap[uid]; !ok {
			fail(ctx, "user not exist")
			return
		}
		friends := userMap[uid].GetFriends()
		ctx.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"friends": friends,
		})
	}
}

// 通过POST上传文件
//
// URL:/file/upload/用户名/文件名
//
// body为上传文件的二进制
//
// 返回:Json{"status", "reason"}
func FileUploadHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user_id := ctx.Param("user")
		suffix := ctx.Param("path")
		userLock.Lock()
		// 判断用户是否存在
		u := userMap[user_id]
		if u == nil {
			userLock.Unlock()
			fail(ctx, "user not exist")
			return
		}
		userLock.Unlock()
		if _, ok := fileMap[user_id+"/"+suffix]; ok {
			fail(ctx, "file existed, delete firse")
			return
		}

		space := u.GetDisk() - u.GetUseddisk()
		if space < ctx.Request.ContentLength {
			fail(ctx, "no enough space")
			return
		}

		// 调用方法，上传文件
		ctl := &file.FileController{}
		ctl.SetSrv(file.FileServiceImpl{})
		f, err := ctl.UploadFile(user_id, suffix, ctx.Request, db)
		if err != nil {
			fail(ctx, err.Error())
			return
		}

		// 上传成功，更新文件拥有者的哈希表,更新文件哈希表
		fileLock.Lock()
		defer fileLock.Unlock()
		fileOwnerMap[f.Uploader] = append(fileOwnerMap[f.Uploader], f)
		fileMap[f.Path] = f
		u.SetUseddisk(u.GetUseddisk() + ctx.Request.ContentLength)
		u.SetFilenum(u.GetFilenum() + 1)
		db.Model(u).Updates(u)
		ctx.JSON(http.StatusOK, gin.H{
			"status": "success",
			// "uploader": user_id,
			// "file":     suffix,
		})
	}
}

// 更新文件的分享目标
//
// 输入:Json{"user_id", "target", "path"}
//
// 返回:Json{"status", "reason"}
func FileTargetHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var msg TargetMsg
		body, _ := ioutil.ReadAll(ctx.Request.Body)
		json.Unmarshal(body, &msg)
		fileLock.Lock()
		defer fileLock.Unlock()

		f := fileMap[msg.Path]
		if f == nil || f.Uploader != msg.UserID {
			fail(ctx, "user doesn't own this file")
			return
		}

		u := userMap[msg.UserID]
		if u == nil {
			fail(ctx, "user not exist")
			return
		}

		ctl := &file.FileController{}
		ctl.SetSrv(file.FileServiceImpl{})

		// 移除原本target哈希表的该文件
		oldTarget := f.GetTarget()
		for _, uid := range oldTarget {
			var i int
			for i = 0; i < len(fileOwnerMap[uid]); i++ {
				if fileOwnerMap[uid][i].Path == msg.Path {
					break
				}
			}
			if i >= len(fileOwnerMap[uid]) {
				continue
			}
			fileOwnerMap[uid] = append(fileOwnerMap[uid][:i], fileOwnerMap[uid][i+1:]...)
		}

		// 从用户的好友列表中,获取在target中的好友
		rawTarget := strings.Split(msg.Target, ",")
		friends := u.GetFriends()
		realTarget := make([]string, 0)
		for _, t := range rawTarget {
			if len(t) > 0 && t != u.Id {
				for _, v := range friends {
					if t == v {
						realTarget = append(realTarget, t)
						break
					}
				}
			}
		}
		err := ctl.UpdateTarget(f, strings.Join(realTarget, ","), db)
		if err != nil {
			fail(ctx, err.Error())
			return
		}

		// 更新分享目标可以获取的文件列表
		for _, uid := range realTarget {
			fileOwnerMap[uid] = append(fileOwnerMap[uid], f)
		}

		ctx.JSON(http.StatusOK, gin.H{
			"status": "success",
		})
	}
}

func FileOwnerHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data":   fileOwnerMap,
		})
	}
}

// 下载文件
//
// 输入:Json{"user_id", "path"}
//
// 输出:文件二进制流
func FileDownloadHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var msg TargetMsg
		body, _ := ioutil.ReadAll(ctx.Request.Body)
		json.Unmarshal(body, &msg)

		f := fileMap[msg.Path]
		if f == nil {
			fail(ctx, "file not exist")
			return
		}

		ctl := &file.FileController{}
		ctl.SetSrv(file.FileServiceImpl{})
		err := ctl.DownloadFile(f, msg.UserID, ctx)
		if err != nil {
			fail(ctx, err.Error())
			return
		}
	}
}

// 删除上传的文件
//
// 输入:Json{"user_id", "path"}
//
// 输出:Json{"status", "reason"}
func FileDeleteHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var msg TargetMsg
		body, _ := ioutil.ReadAll(ctx.Request.Body)
		json.Unmarshal(body, &msg)

		// 检验用户参数
		u := userMap[msg.UserID]
		if u == nil {
			fail(ctx, "user not exist")
			return
		}

		// 检验文件参数
		f := fileMap[msg.Path]
		if f == nil {
			fail(ctx, "file not exist")
			return
		}

		// 调用删除服务
		ctl := &file.FileController{}
		ctl.SetSrv(file.FileServiceImpl{})
		err := ctl.DeleteFile(f, msg.UserID, db)
		if err != nil {
			fail(ctx, err.Error())
			return
		}

		fileLock.Lock()
		defer fileLock.Unlock()

		// 从内存中的Map删除
		target := f.GetTarget()
		target = append(target, f.GetUploader())
		for _, uid := range target {
			var i int
			for i = 0; i < len(fileOwnerMap[uid]); i++ {
				if fileOwnerMap[uid][i].Path == msg.Path {
					break
				}
			}
			if i >= len(fileOwnerMap[uid]) {
				continue
			}
			fileOwnerMap[uid] = append(fileOwnerMap[uid][:i], fileOwnerMap[uid][i+1:]...)
		}

		// 更新上传者的文件数和空间
		u.SetUseddisk(u.GetUseddisk() - f.GetConsume())
		u.SetFilenum(u.GetFilenum() - 1)
		db.Model(u).Updates(u)

		// 从内存中删除
		delete(fileMap, f.GetPath())
		ctx.JSON(http.StatusOK, gin.H{
			"status": "success",
			"reason": msg.Path,
		})
	}
}
