package pdf

import (
	"bytes"
	"net/http"
	"os"
	"path"

	"github.com/Datosystem/go_api_core/controller"
	"github.com/Datosystem/go_api_core/model"
	"github.com/gin-gonic/gin"
	"github.com/phpdave11/gofpdf"
)

func AddPrintRoutes(ctrl controller.CRUDSController, name string, printFunc func(*gin.Context) *gofpdf.Fpdf, pathFunc, fileFunc func(*gin.Context) string) {
	ctrl.AddRoute(http.MethodGet, name, model.PermissionsGet(ctrl.GetModel()), GetReadWriteHandler(printFunc, pathFunc, fileFunc))
	ctrl.AddRoute(http.MethodPost, name, model.PermissionsPost(ctrl.GetModel()), GetWriteHandler(printFunc, pathFunc, fileFunc))
}

func GetPrintHandler(printFunc func(*gin.Context) *gofpdf.Fpdf, fileName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		pdf := printFunc(c)
		if controller.AbortIfError(c, pdf.Error()) {
			return
		}
		fileBuffer := new(bytes.Buffer)
		pdf.Output(fileBuffer)
		c.DataFromReader(http.StatusOK, int64(fileBuffer.Len()), "application/pdf", fileBuffer, map[string]string{
			"Content-Description":       "File Transfer",
			"Content-Transfer-Encoding": "binary",
			"Content-Disposition":       `attachment; filename="` + fileName + `"`,
		})
	}
}

func GetReadWriteHandler(printFunc func(*gin.Context) *gofpdf.Fpdf, pathFunc, fileFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		pdfPath := pathFunc(c)
		pdfFile := fileFunc(c)
		info, err := os.Stat(path.Join(pdfPath, pdfFile))
		if os.IsNotExist(err) {
			GetWriteHandler(printFunc, pathFunc, fileFunc)(c)
			info, err = os.Stat(path.Join(pdfPath, pdfFile))
		}
		if c.IsAborted() {
			return
		}
		file, err := os.Open(path.Join(pdfPath, pdfFile))
		if controller.AbortIfError(c, err) {
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

func GetWriteHandler(printFunc func(*gin.Context) *gofpdf.Fpdf, pathFunc, fileFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := printFunc(c)
		if controller.AbortIfError(c, p.Error()) {
			return
		}
		WritePdf(c, p, pathFunc(c), fileFunc(c))
	}
}
