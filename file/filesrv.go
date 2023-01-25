package file

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type IFileService interface {
	// 更新文件的分享目标
	UpdateTarget(*File, string, *gorm.DB) error
	// 上传文件
	UploadFile(string, string, *http.Request, *gorm.DB) (*File, error)
	// 下载文件
	DownloadFile(*File, string, *gin.Context) error
	// 删除文件
	DeleteFile(*File, string, *gorm.DB) error
}

type FileServiceImpl struct{}

// 更新文件分享目标
func (fi FileServiceImpl) UpdateTarget(f *File, target string, db *gorm.DB) error {
	return f.SetTarget(target, db)
}

// 上传文件
//
// 创建文件夹、读文件数据到内存、创建文件、写文件、更新文件数据库
func (fi FileServiceImpl) UploadFile(userId, fileName string, req *http.Request, db *gorm.DB) (*File, error) {
	// 创建对应用户的文件夹
	err := os.Mkdir("./storage/"+userId, 0777)
	if err != nil && !strings.Contains(err.Error(), "file exists") {
		return nil, err
	}

	// 读取上传的文件数据到内存
	data := make([]byte, req.ContentLength)
	_, err = io.ReadFull(req.Body, data)
	if err != nil {
		return nil, fmt.Errorf("%v when reading file data", err)
	}

	// 创建对应文件
	path := strings.Join([]string{"./storage", userId, fileName}, "/")
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("%v when creating uploaded file", err)
	}
	defer f.Close()

	// 写入数据
	_, err = f.Write(data)
	if err != nil {
		return nil, fmt.Errorf("%v when writing data", err)
	}

	res := &File{Path: userId + "/" + fileName, Uploader: userId, Target: "", Consume: req.ContentLength}
	db.Create(res)
	return res, nil
}

func (fi FileServiceImpl) DownloadFile(f *File, userId string, ctx *gin.Context) error {
	// 用户不是上传者且不是该文件分享的目标
	if userId != f.GetUploader() {
		find := false
		target := f.GetTarget()
		for _, t := range target {
			if userId == t {
				find = true
				break
			}
		}
		if !find {
			return fmt.Errorf("user is not target")
		}
	}
	ctx.File("./storage/" + f.Path)
	return nil
}

func (fi FileServiceImpl) DeleteFile(f *File, user_id string, db *gorm.DB) error {
	if f.GetUploader() != user_id {
		return fmt.Errorf("user is not uploader")
	}
	db.Where("file_path", f.GetPath()).Delete(&File{})
	if db.Error != nil {
		return db.Error
	}
	err := os.Remove("./storage/" + f.GetPath())
	return err
}
