package controller

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Datosystem/go_api_core/message"
	"github.com/Datosystem/go_api_core/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RemoveParentPaths(path string) string {
	return filepath.Clean(strings.ReplaceAll(path, "..", ""))
}

func Folder(pathFunc func(*gin.Context) string) func(*gin.Context) {
	return func(c *gin.Context) {
		type fileInfo struct {
			NAME          string
			PATH          string
			DATE_MODIFIED time.Time
			SIZE          string
		}

		filesDetailed := []fileInfo{}
		files := []string{}

		detailed := c.Query("detailed")

		basePath := pathFunc(c)
		err := filepath.Walk(basePath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				trimmedPath := strings.TrimPrefix(path, basePath+"/")
				trimmedPathRight := strings.TrimRight(path, trimmedPath)

				size := info.Size()

				unit := "B"

				if size >= 1<<30 {
					size /= 1 << 30
					unit = "GB"
				} else if size >= 1<<20 {
					size /= 1 << 20
					unit = "MB"
				} else if size >= 1<<10 {
					size /= 1 << 10
					unit = "KB"
				}

				if detailed != "" {
					filesDetailed = append(filesDetailed, fileInfo{
						NAME:          trimmedPath,
						PATH:          trimmedPathRight,
						DATE_MODIFIED: info.ModTime(),
						SIZE:          fmt.Sprintf("%d %s", size, unit),
					})
				} else {
					files = append(files, trimmedPath)
				}
			}
			return nil
		})

		if err != nil {
			fmt.Printf("Error walking the path %q: %v\n", basePath, err)
		}
		if detailed != "" {
			c.JSON(http.StatusOK, gin.H{"files": filesDetailed})
		} else {
			c.JSON(http.StatusOK, gin.H{"files": files})
		}
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

func CRDController() {
	/*

	 */

	// TODO: Registrare funzioni che gestiscono GET all, GET singolo, POST, DELETE per la cartella specificata, le route disponibili sempre con la stringa CRD
	// TODO: Aggiungere una PermissionFunc specificata dall'utente per ogni metodo (GET, POST e DELETE)
	// TODO: Creare permission func customizzata in base alla risorsa padre (ottenibile da controller.GetModel)
	// TODO: Controllare, nella permission func customizzata, se il modello ha la funzione DefaultConditions, se si eseguire una query COUNT con esse per determinare se l'utente ha accesso alla risorsa base
}

func CheckResourceAvailable(db *gorm.DB, mdl any) bool {
	if condMdl, ok := mdl.(model.ConditionsModel); ok {
		tx := db.Model(mdl)
		table := mdl.(model.TableModel).TableName()
		query, args := condMdl.DefaultConditions(db, table)
		if query != "" {
			tx = tx.Where("("+query+")", args...)
		}
		var count int64
		tx.Count(&count)
		if count == 0 {
			return false
		}
	}

	return true
}

func GetAllFileInFolder(pathFunc func(*gin.Context) string) func(c *gin.Context) {
	return func(c *gin.Context) {

		path := RemoveParentPaths(pathFunc(c))
		dirCon, err := os.ReadDir(path)
		if err != nil {
			message.Forbidden(c).Abort(c)
			return
		}
		arrayNames := []string{}
		for i := range dirCon {

			if !dirCon[i].IsDir() {
				name := dirCon[i].Name()
				arrayNames = append(arrayNames, filepath.Join(path, name))
			}
		}
		c.JSON(http.StatusOK, arrayNames)
	}
}

type FileSystemPermissions struct {
	Get        model.PermissionFunc
	Post       model.PermissionFunc
	GetFile    model.PermissionFunc
	Delete     model.PermissionFunc
	Conditions model.PermissionFunc
}

func FileSystem(ctrl AddRouter, apiPath string, filePath func(c *gin.Context) string, permissions FileSystemPermissions) {
	ctrl.AddRoute(http.MethodGet, apiPath, model.PermissionsMerge(permissions.Get, permissions.Conditions), Folder(filePath))
	ctrl.AddRoute(http.MethodPost, apiPath, model.PermissionsMerge(permissions.Post, permissions.Conditions), PostFile(filePath))
	ctrl.AddRoute(http.MethodGet, apiPath+"/:fileName", model.PermissionsMerge(permissions.GetFile, permissions.Conditions), GetFile(filePath, func(c *gin.Context) string { return c.Param("fileName") }))
	ctrl.AddRoute(http.MethodDelete, apiPath+"/:fileName", model.PermissionsMerge(permissions.Delete, permissions.Conditions), DeleteFile(filePath, func(c *gin.Context) string { return c.Param("fileName") }))
}

func DefaultFileSystemPermissions(ctrl GetModeler) FileSystemPermissions {
	mdl := ctrl.GetModel()
	fsPermissions := FileSystemPermissions{
		Get:     model.PermissionsGet(mdl),
		Post:    model.PermissionsPost(mdl),
		GetFile: model.PermissionsGet(mdl),
		Delete:  model.PermissionsDelete(mdl),
	}
	if condMdl, ok := mdl.(model.ConditionsModel); ok {
		fsPermissions.Conditions = func(c *gin.Context) message.Message {
			db := c.MustGet("db").(*gorm.DB)
			primaries := map[string]interface{}{}
			GetPathParams(c, ctrl.NewModel(), GetPrimaryFields(ctrl.GetModelType()), &primaries)

			tx := db.Model(mdl).Where(primaries)
			table := mdl.(model.TableModel).TableName()
			query, args := condMdl.DefaultConditions(db, table)
			if query != "" {
				tx = tx.Where("("+query+")", args...)
			}

			var count int64
			tx.Debug().Count(&count)

			if count == 0 {
				return message.ItemNotFound(c)
			}
			return nil
		}
	}
	return fsPermissions
}
