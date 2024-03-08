package controller

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/Datosystem/go_api_core/app"
	"github.com/Datosystem/go_api_core/datatypes"
	"github.com/Datosystem/go_api_core/message"
	"github.com/Datosystem/go_api_core/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type AddRouter interface {
	AddRoute(method string, name string, permissionsFunc model.PermissionFunc, handlersFunc ...gin.HandlerFunc)
}

type GetModeler interface {
	GetModel() any
	GetModelType() reflect.Type

	NewModel() any
	NewSliceOfModel() any
}

type CRUDSController interface {
	AddRouter
	GetModeler

	SetBasePath(basePath string)
	SetEndpointIfAbsent(name string)
	GetEndpoint() string
	GetEndpointPath() string

	Get(c *gin.Context)
	GetOne(c *gin.Context)
	GetStructure(c *gin.Context)
	GetRelStructure(c *gin.Context)
	Post(c *gin.Context)
	Patch(c *gin.Context)
	PatchMany(c *gin.Context)
	Delete(c *gin.Context)

	CanImport() bool

	AddCustomRoutes()
	AdditionalModels() []reflect.Type
	GetRoutes() []Route
}

type Route struct {
	Method          string
	Name            string
	PermissionsFunc model.PermissionFunc
	HandlerFuncs    []gin.HandlerFunc
}

type Controller struct {
	Model    interface{}
	BasePath string
	Endpoint string
	Routes   []Route
}

func (r Controller) NewModel() interface{} {
	return reflect.New(r.GetModelType()).Interface()
}

func (r Controller) NewSliceOfModel() interface{} {
	return reflect.New(reflect.SliceOf(r.GetModelType())).Interface()
}

func (r Controller) GetModel() interface{} {
	return r.Model
}

func (r Controller) GetModelType() reflect.Type {
	if r.Model == nil {
		return nil
	}
	return reflect.Indirect(reflect.ValueOf(r.Model)).Type()
}

func (r *Controller) SetBasePath(basePath string) {
	r.BasePath = basePath
}

func (r *Controller) SetEndpointIfAbsent(name string) {
	if len(r.Endpoint) > 0 {
		return
	}
	r.Endpoint = name
}

func (r Controller) GetEndpoint() string {
	return r.Endpoint
}

func (r Controller) GetEndpointPath() string {
	return r.BasePath + "/" + r.Endpoint
}

func (r Controller) Get(c *gin.Context) {
	HandleGet(c, c.MustGet("db").(*gorm.DB), map[string]interface{}{}, r.NewModel())
}

func HandleGet(c *gin.Context, db *gorm.DB, primaries map[string]interface{}, model any) {
	args := QueryMapArgs{
		Sel:       c.Query("sel"),
		Rel:       c.Query("rel"),
		Params:    c.Query("params"),
		P:         c.Query("p"),
		PagStart:  c.Query("pagStart"),
		PagEnd:    c.Query("pagEnd"),
		Ord:       c.Query("ord"),
		Primaries: primaries,
		Model:     model,
	}
	err := QueryMap(c, db, &args, QueryMapConfig{})
	if AbortIfError(c, err) {
		return
	}
	WriteQueryMapResult(c, &args)
}

func WriteQueryMapResult(c *gin.Context, args *QueryMapArgs) {
	if !c.IsAborted() {
		c.Header("X-Total-Count", strconv.Itoa(int(args.Count)))
		var link string
		if ShouldPaginate(args.PagStart, args.PagEnd) {
			limit := GetLimit(args.PagStart, args.PagEnd)
			start := GetOffset(args.PagStart) + limit
			end := start + limit
			if int64(end) < args.Count {
				var params []string
				for key, values := range c.Request.URL.Query() {
					if key == "pagStart" {
						params = append(params, key+"="+strconv.Itoa(start))
					} else if key == "pagEnd" {
						params = append(params, key+"="+strconv.Itoa(end))
					} else {
						params = append(params, key+"="+strings.Join(values, "&"+key+"="))
					}
				}
				link = c.FullPath() + "?" + strings.Join(params, "&")
				c.Header("Link", link)
			}
		}
		switch c.GetHeader("Accept") {
		case "application/csv", "text/csv":
			c.Header("Content-Type", c.GetHeader("Accept")+"; charset=utf-8")
			c.Header("Content-Disposition", "attachment; filename=data.csv")
			c.Status(http.StatusOK)

			// TODO: Done but needs better checking; sorting based on the input select
			// TODO: Manage the CSV in the correct order

			var csvData [][]string
			tmz := c.Request.Header.Get("Timezone")
			var heading []string
			for _, f := range args.Info.Fields {
				heading = append(heading, f.Name)
			}
			for key := range args.Info.Nested {
				heading = append(heading, key)
			}
			// for i := range heading {
			// 	if strings.Contains(heading[i], "AS") {
			// 		s := strings.Split(heading[i], "AS")
			// 		heading[i] = strings.TrimSpace(s[len(s)-1])
			// 	}
			// }
			l := len(args.Result)
			if l > 0 {

				csvData = append(csvData, heading)

				for i := 0; i < l; i++ {
					item := args.Result[i]
					var row []string
					for _, f := range args.Info.Fields {
						r := reflect.ValueOf(item[f.Name])
						if r.IsValid() && !r.IsZero() && !r.IsNil() {
							t := reflect.Indirect(r)
							if t.Type().Kind() == reflect.Ptr {
								t = t.Elem()
							}
							f := t.Interface()
							if f != nil && f != "" {
								if date, ok := f.(datatypes.Date); ok {
									row = append(row, time.Time(date).Format("02/01/2006"))
								} else if datetime, ok := f.(datatypes.Datetime); ok {
									if c.GetHeader("Only-Date") == "" {
										loc, _ := time.LoadLocation(tmz)
										row = append(row, time.Time(datetime).In(loc).Format("02/01/2006 15:04"))
									} else {
										row = append(row, time.Time(datetime).Format("02/01/2006"))
									}
								} else if _, ok := f.(datatypes.RoundedFloat); ok {
									row = append(row, strings.ReplaceAll(fmt.Sprint(f), ".", ","))
								} else if _, ok := f.(float32); ok {
									row = append(row, strings.ReplaceAll(fmt.Sprint(f), ".", ","))
								} else if _, ok := f.(float64); ok {
									row = append(row, strings.ReplaceAll(fmt.Sprint(f), ".", ","))
								} else {
									if _, ok := f.(string); ok {
										row = append(row, fmt.Sprintf("%s", f))
									} else {
										row = append(row, fmt.Sprint(f))
									}
								}
							} else {
								row = append(row, "")
							}
						} else {
							row = append(row, "")
						}
					}
					for key := range args.Info.Nested {
						data, err := json.Marshal(item[key])
						if err != nil {
							fmt.Println(err)
						}
						row = append(row, string(data))
					}
					csvData = append(csvData, row)
				}
			}
			if err := csv.NewWriter(c.Writer).WriteAll(csvData); err != nil {
				AbortWithError(c, err)
			}
		case "application/xml", "text/xml":
			if reflect.TypeOf(args.Result).Name() == "" || len(c.Query("wrap")) > 0 {
				c.XML(http.StatusOK, Response{Data: args.Result, Next: link, Count: args.Count})
			} else {
				c.XML(http.StatusOK, args.Result)
			}
		default:
			var result any = args.Result
			if len(args.Primaries) != 0 {
				result = args.Result[0]
				// TODO: It might be advisable to set Count to 1 in this situation
			}
			if len(c.Query("wrap")) > 0 {
				c.JSON(http.StatusOK, Response{Data: result, Next: link, Count: args.Count})
			} else {
				c.JSON(http.StatusOK, result)
			}
		}
	}
}

func WriteDataWithCount(c *gin.Context, pagStart, pagEnd string, data any, count int64) {
	if !c.IsAborted() {
		c.Header("X-Total-Count", strconv.Itoa(int(count)))
		var link string
		if ShouldPaginate(pagStart, pagEnd) {
			limit := GetLimit(pagStart, pagEnd)
			start := GetOffset(pagStart) + limit
			end := start + limit
			if int64(end) < count {
				var params []string
				for key, values := range c.Request.URL.Query() {
					if key == "pagStart" {
						params = append(params, key+"="+strconv.Itoa(start))
					} else if key == "pagEnd" {
						params = append(params, key+"="+strconv.Itoa(end))
					} else {
						params = append(params, key+"="+strings.Join(values, "&"+key+"="))
					}
				}
				link = c.FullPath() + "?" + strings.Join(params, "&")
				c.Header("Link", link)
			}
		}
		switch c.GetHeader("Accept") {
		case "application/csv", "text/csv":
			c.Header("Content-Type", c.GetHeader("Accept")+"; charset=utf-8")
			c.Header("Content-Disposition", "attachment; filename=data.csv")
			c.Status(http.StatusOK)

			var csvData [][]string
			v := reflect.ValueOf(data).Elem()
			t := v.Type()
			if t.Kind() != reflect.Slice {
				t = reflect.SliceOf(t)
				v = reflect.Append(reflect.New(t).Elem(), v)
			}

			l := v.Len()
			if l > 0 {

				t = t.Elem()
				if t.Kind() == reflect.Ptr {
					t = t.Elem()
				}

				var row []string
				for j := 0; j < t.NumField(); j++ {
					row = append(row, t.Field(j).Name)
				}
				csvData = append(csvData, row)

				for i := 0; i < l; i++ {
					item := v.Index(i).Elem()
					ti := item.Type()
					var row []string
					for j := 0; j < ti.NumField(); j++ {
						f := item.Field(j)
						if f.IsValid() && !f.IsZero() {
							if f.Type().Kind() == reflect.Ptr {
								f = f.Elem()
							}
							if f.Type().Kind() == reflect.Slice || f.Type().Kind() == reflect.Array {
								v, _ := json.Marshal(f.Interface())
								row = append(row, string(v))
							} else if marshaler, ok := f.Interface().(json.Marshaler); ok {
								v, _ := marshaler.MarshalJSON()
								row = append(row, strings.TrimSuffix(strings.TrimPrefix(string(v), "\""), "\""))
							} else if stringer, ok := f.Interface().(fmt.Stringer); ok {
								fmt.Println(stringer.String())
								row = append(row, stringer.String())
							} else {
								fmt.Println(f.Interface())
								row = append(row, fmt.Sprintf("%v", f.Interface()))
							}
						} else {
							row = append(row, "")
						}
					}
					csvData = append(csvData, row)
				}
			}
			if err := csv.NewWriter(c.Writer).WriteAll(csvData); err != nil {
				AbortWithError(c, err)
			}
		case "application/xml", "text/xml":
			if reflect.TypeOf(data).Name() == "" || len(c.Query("wrap")) > 0 {
				c.XML(http.StatusOK, Response{Data: data, Next: link, Count: count})
			} else {
				c.XML(http.StatusOK, data)
			}
		default:
			if len(c.Query("wrap")) > 0 {
				c.JSON(http.StatusOK, Response{Data: data, Next: link, Count: count})
			} else {
				c.JSON(http.StatusOK, data)
			}
		}
	}
}

func (r Controller) GetOne(c *gin.Context) {
	primaries := map[string]interface{}{}
	GetPathParams(c, r.NewModel(), GetPrimaryFields(r.GetModelType()), &primaries)
	if c.IsAborted() {
		return
	}
	HandleGet(c, c.MustGet("db").(*gorm.DB), primaries, r.NewModel())
}

func (r Controller) GetStructure(c *gin.Context) {
	relations := GetRelations(c)
	splittedRelations := [][]string{}
	for _, rel := range relations {
		splittedRelations = append(splittedRelations, strings.Split(rel, "."))
	}

	modelSchema, err := schema.Parse(r.GetModel(), &sync.Map{}, app.DB.NamingStrategy)
	if err != nil {
		message.InternalServerError(c).Abort(c)
		return
	}

	c.JSON(http.StatusOK, GetStructInfo(c, modelSchema, splittedRelations))
}

func (r Controller) GetRelStructure(c *gin.Context) {
	modelSchema, err := schema.Parse(r.GetModel(), &sync.Map{}, app.DB.NamingStrategy)
	if err != nil {
		message.InternalServerError(c).Abort(c)
		return
	}

	pieces := strings.Split(c.Param("rel"), ".")
	relSchema := modelSchema
	for i, piece := range pieces {
		if rel, ok := relSchema.Relationships.Relations[piece]; ok {
			relSchema = rel.FieldSchema
			if msg := model.PermissionsGet(reflect.New(relSchema.ModelType).Interface())(c); msg != nil {
				message.UnauthorizedRelations(c, strings.Join(pieces[:i+1], ".")).Add(msg).Abort(c)
				return
			}
		} else {
			message.InvalidRelations(c, strings.Join(pieces[:i+1], ".")).Abort(c)
			return
		}
	}

	c.JSON(http.StatusOK, GetStructInfo(c, relSchema, [][]string{}))
}

func (r Controller) Post(c *gin.Context) {
	jsonData, err := c.GetRawData()

	if err != nil || len(jsonData) == 0 {
		message.InvalidJSON(c).Abort(c)
		return
	}

	if jsonData[0] == '[' {
		model := r.NewSliceOfModel()
		LoadModel(c, jsonData, model)
		ValidateModels(c, model)
		CreateToDb(c, c.MustGet("db").(*gorm.DB), model)
	} else {
		model := r.NewModel()
		LoadModel(c, jsonData, model)
		ValidateModel(c, model)
		CreateToDb(c, c.MustGet("db").(*gorm.DB), model)
	}
}

func (r Controller) Patch(c *gin.Context) {
	model := r.NewModel()
	jsonMap := make(map[string]interface{})
	jsonData, _ := c.GetRawData()
	modelType := r.GetModelType()
	primaryFields := GetPrimaryFields(modelType)

	LoadModel(c, jsonData, model)
	GetPathParams(c, model, primaryFields, model)
	LoadAndValidateMap(c, jsonData, jsonMap, modelType)
	GetPathParams(c, model, primaryFields, &jsonMap)
	UpdateToDb(c, model, jsonMap)
}

func (r Controller) PatchMany(c *gin.Context) {
	modelSlice := r.NewSliceOfModel()
	jsonMaps := []map[string]interface{}{}
	jsonData, _ := c.GetRawData()
	modelType := r.GetModelType()

	LoadModel(c, jsonData, modelSlice)
	LoadAndValidateMaps(c, jsonData, &jsonMaps, modelType)
	ValidateMapsPrimaries(c, jsonMaps, GetPrimaryFields(modelType))
	if c.IsAborted() {
		return
	}
	if len(jsonMaps) > 0 {
		db := c.MustGet("db").(*gorm.DB).Session(&gorm.Session{CreateBatchSize: 50})

		modelSliceVal := reflect.ValueOf(modelSlice).Elem()

		modelSchema, err := schema.Parse(modelSliceVal.Index(0), &sync.Map{}, db.NamingStrategy)
		if err != nil {
			message.InternalServerError(c).Abort(c)
			return
		}

		checked := map[string]struct{}{}
		for i := range jsonMaps {
			msg := CheckModelPermissions(c, modelSliceVal.Index(i), modelSchema, checked, true)
			if msg != nil {
				msg.Abort(c)
				return
			}
		}

		db.Session(&gorm.Session{FullSaveAssociations: true}).Transaction(func(tx *gorm.DB) error {
			for i, values := range jsonMaps {
				modelVal := modelSliceVal.Index(i).Addr()
				e := DeleteRelations(c, tx, modelVal, modelSchema)
				if e != nil {
					return e
				}
				if tx.Error != nil {
					return tx.Error
				}
				res := tx.Model(modelVal.Interface()).Updates(values)
				if res.Error != nil {
					return res.Error
				}
			}

			return nil
		})
	}

	c.JSON(http.StatusOK, modelSlice)
}

func (r Controller) Delete(c *gin.Context) {
	primaryFields := GetPrimaryFields(r.GetModelType())
	models := []interface{}{}
	PathParamsToModels(c, r.GetModelType(), primaryFields, &models)
	DeleteFromDb(c, models)
}

func (r *Controller) CanImport() bool {
	return false
}

func (r *Controller) AddRoute(method string, name string, permissionsFunc model.PermissionFunc, handlersFunc ...gin.HandlerFunc) {
	r.Routes = append(r.Routes, Route{method, name, permissionsFunc, handlersFunc})
}

func (r Controller) AddCustomRoutes() {}

func (r Controller) AdditionalModels() []reflect.Type {
	return []reflect.Type{}
}

func (r Controller) GetRoutes() []Route {
	return r.Routes
}
