package controller

import (
	"bytes"
	"net/http"
	"os"
	"path"

	"github.com/Datosystem/go_api_core/message"
	"github.com/Datosystem/go_api_core/model"
	"github.com/gin-gonic/gin"
	"github.com/phpdave11/gofpdf"
	"gorm.io/gorm"
)

func PrintHandler(printFunc func(*gin.Context) *gofpdf.Fpdf, fileFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		pdf := printFunc(c)
		pdfFile := fileFunc(c)
		if AbortIfError(c, pdf.Error()) {
			return
		}
		fileBuffer := new(bytes.Buffer)
		pdf.Output(fileBuffer)
		c.DataFromReader(http.StatusOK, int64(fileBuffer.Len()), "application/pdf", fileBuffer, map[string]string{
			"Content-Description":       "File Transfer",
			"Content-Transfer-Encoding": "binary",
			"Content-Disposition":       `attachment; filename="` + pdfFile + `"`,
		})
	}
}

func PrintReadWriteHandler(printFunc func(*gin.Context) *gofpdf.Fpdf, pathFunc, fileFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		pdfPath := pathFunc(c)
		pdfFile := fileFunc(c)
		pathGetter := func(c *gin.Context) string { return pdfPath }
		fileGetter := func(c *gin.Context) string { return pdfFile }

		if !c.GetBool("SkipPrintWriteHandler") {
			_, err := os.Stat(path.Join(pdfPath, pdfFile))
			if os.IsNotExist(err) {
				PrintWriteHandler(printFunc, pathGetter, fileGetter)(c)
				if c.IsAborted() {
					return
				}
			}
		}
		PrintReadHandler(pathGetter, fileGetter)(c)
	}
}

func PrintReadHandler(pathFunc, fileFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		pdfPath := pathFunc(c)
		pdfFile := fileFunc(c)

		info, err := os.Stat(path.Join(pdfPath, pdfFile))
		if AbortIfError(c, err) {
			return
		}

		file, err := os.Open(path.Join(pdfPath, pdfFile))
		if AbortIfError(c, err) {
			return
		}
		defer file.Close()

		fileBuffer := new(bytes.Buffer)
		fileBuffer.ReadFrom(file)
		c.DataFromReader(http.StatusOK, int64(fileBuffer.Len()), "application/pdf", fileBuffer, map[string]string{
			"Content-Description":       "File Transfer",
			"Content-Transfer-Encoding": "binary",
			"Content-Disposition":       `attachment; filename="` + pdfFile + `"`,
			"Data-Modifica":             info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}
}

func PrintWriteHandler(printFunc func(*gin.Context) *gofpdf.Fpdf, pathFunc, fileFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := printFunc(c)
		if AbortIfError(c, p.Error()) {
			return
		}
		writePdf(c, p, pathFunc(c), fileFunc(c))
	}
}

func writePdf(c *gin.Context, p *gofpdf.Fpdf, folder, file string) {
	if p.Error() != nil {
		AbortWithError(c, p.Error())
		return
	}
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		os.MkdirAll(folder, os.ModePerm)
	}
	err := p.OutputFileAndClose(path.Join(folder, file))
	if err != nil {
		AbortWithError(c, p.Error())
	}
}

func Print(ctrl CRUDSController, name string, printFunc func(*gin.Context) *gofpdf.Fpdf, pathFunc, fileFunc func(*gin.Context) string, permissions FileSystemPermissions) {
	ctrl.AddRoute(http.MethodGet, name, model.PermissionsMerge(permissions.Get, permissions.Conditions), PrintReadWriteHandler(printFunc, pathFunc, fileFunc))
	ctrl.AddRoute(http.MethodPost, name, model.PermissionsMerge(permissions.Post, permissions.Conditions), PrintWriteHandler(printFunc, pathFunc, fileFunc))
}

func DefaultPrintPermissions(ctrl GetModeler) FileSystemPermissions {
	mdl := ctrl.GetModel()
	printPermissions := FileSystemPermissions{
		Get:  model.PermissionsGet(mdl),
		Post: model.PermissionsPost(mdl),
	}
	if condMdl, ok := mdl.(model.ConditionsModel); ok {
		printPermissions.Conditions = func(c *gin.Context) message.Message {
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
	return printPermissions
}
