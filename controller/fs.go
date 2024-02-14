package controller

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Datosystem/go_api_core/message"
	"github.com/Datosystem/go_api_core/model"
	"github.com/gin-gonic/gin"
)

func RemoveParentPaths(path string) string {
	return filepath.Clean(strings.ReplaceAll(path, "..", ""))
}

func Folder(pathFunc func(*gin.Context) string) func(*gin.Context) {
	return func(c *gin.Context) {
		path := RemoveParentPaths(pathFunc(c))
		file := filepath.Join(path)
		if !strings.HasPrefix(filepath.Clean(file), path) {
			message.Forbidden(c).Abort(c)
			return
		}
		files, err := os.ReadDir(file)
		if err != nil {
			message.FolderNotFound(c).Abort(c)
			return
		}
		fileNames := []string{}
		for _, f := range files {
			fileNames = append(fileNames, f.Name())
		}
		c.JSON(http.StatusOK, gin.H{"files": fileNames})
	}
}

func GetFile(pathFunc, nameFunc func(*gin.Context) string) func(*gin.Context) {
	return func(c *gin.Context) {
		path := RemoveParentPaths(pathFunc(c))
		name := nameFunc(c)
		file := filepath.Join(path, name)
		if !strings.HasPrefix(filepath.Clean(file), path) {
			message.Forbidden(c).Abort(c)
			return
		}

		if c.Query("download") == "" {
			c.Header("Content-Disposition", "inline; filename="+name)
		} else {
			c.Header("Content-Description", "File Transfer")
			c.Header("Content-Transfer-Encoding", "binary")
			c.Header("Content-Disposition", "attachment; filename="+name)
			c.Header("Content-Type", "application/octet-stream")
		}
		c.File(file)
	}
}

func PostFile(pathFunc func(*gin.Context) string) func(*gin.Context) {
	return func(c *gin.Context) {
		path := RemoveParentPaths(pathFunc(c))
		numeroFiles := c.Query("numeroFiles")

		if len(numeroFiles) > 0 {
			fileLength, err := strconv.Atoi(numeroFiles)
			if err != nil {
				// ... handle error
				panic(err)
			}
			for i := 0; i < fileLength; i++ {
				file, err := c.FormFile("file" + strconv.Itoa(i))
				if err != nil {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
						"message": err,
					})
					log.Println(err)
					return
				}
				newFileName := file.Filename

				if err := c.SaveUploadedFile(file, filepath.Join(path, newFileName)); err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"message": "Unable to save the file",
					})
					return
				}
			}
		} else {
			file, err := c.FormFile("file")
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"message": err,
				})
				log.Println(err)
				return
			}
			newFileName := file.Filename

			if err := c.SaveUploadedFile(file, filepath.Join(path, newFileName)); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"message": "Unable to save the file",
				})
				return
			}
		}
		message.Ok(c).JSON(c)
	}
}

func DeleteFile(pathFunc, nameFunc func(*gin.Context) string) func(*gin.Context) {
	return func(c *gin.Context) {
		path := RemoveParentPaths(pathFunc(c))
		name := nameFunc(c)
		file := filepath.Join(path, name)
		if !strings.HasPrefix(filepath.Clean(file), path) {
			message.Forbidden(c).Abort(c)
			return
		}
		os.Remove(file)
		message.Ok(c).JSON(c)
	}
}

type FileSystemPermissions struct {
	Get     model.PermissionFunc
	Post    model.PermissionFunc
	GetFile model.PermissionFunc
	Delete  model.PermissionFunc
}

func FileSystem(ctrl AddRouter, apiPath string, filePath func(c *gin.Context) string, permissions FileSystemPermissions) {
	ctrl.AddRoute(http.MethodGet, apiPath, permissions.Get, Folder(filePath))
	ctrl.AddRoute(http.MethodPost, apiPath, permissions.Post, PostFile(filePath))
	ctrl.AddRoute(http.MethodGet, apiPath+"/:fileName", permissions.GetFile, GetFile(filePath, func(c *gin.Context) string { return c.Param("fileName") }))
	ctrl.AddRoute(http.MethodDelete, apiPath+"/:fileName", permissions.Delete, DeleteFile(filePath, func(c *gin.Context) string { return c.Param("fileName") }))
}

func DefaultFileSystemPermissions(ctrl GetModeler) FileSystemPermissions {
	return FileSystemPermissions{
		Get:     model.PermissionsGet(ctrl.GetModel()),
		Post:    model.PermissionsPost(ctrl.GetModel()),
		GetFile: model.PermissionsGet(ctrl.GetModel()),
		Delete:  model.PermissionsDelete(ctrl.GetModel()),
	}
}
