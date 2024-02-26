package controller

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/Datosystem/go_api_core/model"

	"github.com/gin-gonic/gin"
)

var ByName = map[string]CRUDSController{}
var ByModel = map[string]CRUDSController{}

func Register(container *gin.RouterGroup, toRegister string, r CRUDSController) *gin.RouterGroup {
	r.SetBasePath(container.BasePath())
	controllerName := reflect.Indirect(reflect.ValueOf(r)).Type().Name()
	r.SetEndpointIfAbsent(FirstLower(controllerName))
	name := r.GetEndpoint()

	if ByName[controllerName] != nil {
		panic("Controller " + name + " already registered!")
	}

	ByName[controllerName] = r

	modelType := r.GetModelType()
	if modelType != nil {
		primaryFields := GetPrimaryFields(r.GetModelType())
		params := ""
		for i, field := range primaryFields {
			if i > 0 {
				params += "/"
			}
			params += ":" + field
		}

		if strings.Contains(toRegister, "C") {
			r.AddRoute(http.MethodPost, "", model.PermissionsPost(r.GetModel()), r.Post)
		}
		if strings.Contains(toRegister, "R") {
			r.AddRoute(http.MethodGet, "", model.PermissionsGet(r.GetModel()), r.Get)
			if len(primaryFields) > 0 {
				r.AddRoute(http.MethodGet, params, model.PermissionsGet(r.GetModel()), r.GetOne)
			}
		}
		if strings.Contains(toRegister, "U") && len(primaryFields) > 0 {
			r.AddRoute(http.MethodPatch, params, model.PermissionsPatch(r.GetModel()), r.Patch)
			r.AddRoute(http.MethodPatch, "", model.PermissionsPatch(r.GetModel()), r.PatchMany)
		}
		if strings.Contains(toRegister, "D") && len(primaryFields) > 0 {
			r.AddRoute(http.MethodDelete, params, model.PermissionsDelete(r.GetModel()), r.Delete)
		}
		if strings.Contains(toRegister, "S") {
			r.AddRoute(http.MethodGet, "structure", model.PermissionsGet(r.GetModel()), r.GetStructure)
			r.AddRoute(http.MethodGet, "structure/:rel", model.PermissionsGet(r.GetModel()), r.GetRelStructure)
		}

		ByModel[modelType.String()] = r
	}

	for _, typ := range r.AdditionalModels() {
		ByModel[typ.String()] = r
	}

	r.AddCustomRoutes()

	grp := container.Group(name)

	for _, route := range r.GetRoutes() {
		funcs := []gin.HandlerFunc{}
		if route.PermissionsFunc != nil {
			funcs = append(funcs, checkPermissions(route.PermissionsFunc))
		}
		funcs = append(funcs, route.HandlerFuncs...)
		grp.Handle(route.Method, route.Name, funcs...)
	}

	return grp
}

func FindControllerByModel(modelType reflect.Type) CRUDSController {
	return ByModel[modelType.String()]
}

func checkPermissions(permissionFunc model.PermissionFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		message := permissionFunc(c)
		if message != nil {
			message.Abort(c)
		}
	}
}
