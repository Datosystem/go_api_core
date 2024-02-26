package model

import (
	"reflect"

	"github.com/Datosystem/go_api_core/app"
	"github.com/Datosystem/go_api_core/message"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/schema"
)

type PermissionFunc func(c *gin.Context) message.Message

type ModelWithPermissionsPrefix interface {
	PermissionsPrefix() string
}

type ModelWithPermissionsGet interface {
	PermissionsGet(c *gin.Context) message.Message
}

type ModelWithPermissionsPost interface {
	PermissionsPost(c *gin.Context) message.Message
}

type ModelWithPermissionsPatch interface {
	PermissionsPatch(c *gin.Context) message.Message
}

type ModelWithPermissionsDelete interface {
	PermissionsDelete(c *gin.Context) message.Message
}

func PermissionsPrefix(model interface{}) string {
	var prefix string
	if prefixModel, ok := model.(ModelWithPermissionsPrefix); ok {
		prefix = prefixModel.PermissionsPrefix()
	} else if tableModel, ok := model.(TableModel); ok {
		prefix = tableModel.TableName()
	} else {
		typ := reflect.TypeOf(model)
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		// Note: this uses the default naming strategy at the moment to keep things simple
		return schema.NamingStrategy{
			NoLowerCase:   true,
			SingularTable: true,
		}.TableName(typ.Name())
	}
	return prefix
}

func PermissionsGet(model interface{}) PermissionFunc {
	if modelPerm, ok := model.(ModelWithPermissionsGet); ok {
		return modelPerm.PermissionsGet
	} else {
		return func(c *gin.Context) message.Message {
			return c.MustGet("s").(*app.Session).CheckOne(c, PermissionsPrefix(model)+"_GET")
		}
	}
}

func PermissionsPost(model interface{}) PermissionFunc {
	if modelPerm, ok := model.(ModelWithPermissionsPost); ok {
		return modelPerm.PermissionsPost
	} else {
		return func(c *gin.Context) message.Message {
			return c.MustGet("s").(*app.Session).CheckOne(c, PermissionsPrefix(model)+"_POST")
		}
	}
}

func PermissionsPatch(model interface{}) PermissionFunc {
	if modelPerm, ok := model.(ModelWithPermissionsPatch); ok {
		return modelPerm.PermissionsPatch
	} else {
		return func(c *gin.Context) message.Message {
			return c.MustGet("s").(*app.Session).CheckOne(c, PermissionsPrefix(model)+"_PATCH")
		}
	}
}

func PermissionsDelete(model interface{}) PermissionFunc {
	if modelPerm, ok := model.(ModelWithPermissionsDelete); ok {
		return modelPerm.PermissionsDelete
	} else {
		return func(c *gin.Context) message.Message {
			return c.MustGet("s").(*app.Session).CheckOne(c, PermissionsPrefix(model)+"_DELETE")
		}
	}
}
