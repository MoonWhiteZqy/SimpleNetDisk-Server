package file

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FileController struct {
	fileservice IFileService
}

func (c *FileController) SetSrv(srv IFileService) {
	c.fileservice = srv
}

func (c *FileController) UpdateTarget(f *File, target string, db *gorm.DB) error {
	return c.fileservice.UpdateTarget(f, target, db)
}

func (c *FileController) UploadFile(userId, fileName string, req *http.Request, db *gorm.DB) (*File, error) {
	return c.fileservice.UploadFile(userId, fileName, req, db)
}

func (c *FileController) DownloadFile(f *File, userId string, ctx *gin.Context) error {
	return c.fileservice.DownloadFile(f, userId, ctx)
}

func (c *FileController) DeleteFile(f *File, user_id string, db *gorm.DB) error {
	return c.fileservice.DeleteFile(f, user_id, db)
}
