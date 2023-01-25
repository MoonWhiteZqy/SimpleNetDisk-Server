package file

import (
	"strings"

	"gorm.io/gorm"
)

type File struct {
	gorm.Model
	Path     string `gorm:"column:file_path"`
	Uploader string `gorm:"column:file_uploader"`
	Target   string `gorm:"column:share_target"`
	Consume  int64  `gorm:"column:file_consume"`
}

func (f *File) GetPath() string {
	return f.Path
}

func (f *File) SetPath(path string) bool {
	f.Path = path
	return true
}

func (f *File) GetUploader() string {
	return f.Uploader
}

func (f *File) SetUploader(user string) bool {
	f.Uploader = user
	return true
}

func (f *File) GetTarget() []string {
	if len(f.Target) == 0 {
		return make([]string, 0)
	}
	return strings.Split(f.Target, ",")
}

func (f *File) SetTarget(target string, db *gorm.DB) error {
	f.Target = target
	db.Model(f).Update("share_target", target)
	return db.Error
}

func (f *File) GetConsume() int64 {
	return f.Consume
}
