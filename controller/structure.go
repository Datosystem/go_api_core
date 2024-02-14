package controller

import (
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/Datosystem/go_api_core/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/schema"
)

type FieldInfo struct {
	Field           string `json:"field"`
	Label           string `json:"label"`
	Descriptive     string `json:"descriptive"`
	Type            string `json:"type"`
	Primary         bool   `json:"primary"`
	Required        bool   `json:"required"`
	UpdateKey       bool   `json:"updateKey"`
	Deprecated      bool   `json:"deprecated"`
	MaxLength       int    `json:"maxLength"`
	RequiredWithout string `json:"requiredWithout"`
	RequiredWith    string `json:"requiredWith"`
	Updatable       bool   `json:"updatable"`
	Creatable       bool   `json:"creatable"`
	Query           bool   `json:"query"`
}

type RelationInfo struct {
	Field      string      `json:"field"`
	Label      string      `json:"label"`
	ForeignKey string      `json:"foreignKey"`
	References string      `json:"references"`
	Type       string      `json:"type"`
	Endpoint   string      `json:"endpoint"`
	Struct     *StructInfo `json:"struct"`
	Updatable  bool        `json:"updatable"`
	Creatable  bool        `json:"creatable"`
}

type StructInfo struct {
	Fields           []FieldInfo              `json:"fields"`
	Relations        []RelationInfo           `json:"relations"`
	UpdateConditions []model.UpdateConditions `json:"updateConditions"`
}

func GetStructInfo(c *gin.Context, schem *schema.Schema, relations [][]string) StructInfo {
	var checkFn = func(fns []func(*schema.Field) bool, sc *schema.Field) bool {
		for _, fn := range fns {
			if !fn(sc) {
				return false
			}
		}
		return true
	}
	checkFieldFns := []func(*schema.Field) bool{}
	checkRelFns := []func(*schema.Field) bool{
		func(f *schema.Field) bool {
			typ := f.IndirectFieldType
			if typ.Kind() == reflect.Slice {
				typ = typ.Elem()
			}
			mdl := reflect.New(typ).Interface()
			msg := model.PermissionsGet(mdl)(c)
			return msg == nil
		},
	}

	if c.Query("w") == "1" {
		// Checks for only writable
		fn := func(f *schema.Field) bool {
			return f.Creatable && f.Updatable
		}
		checkFieldFns = append(checkFieldFns, fn)
		checkRelFns = append(checkRelFns, fn)
	}
	if c.Query("r") == "1" {
		// Checks for only readable
		fn := func(f *schema.Field) bool {
			return f.Readable
		}
		checkFieldFns = append(checkFieldFns, fn)
		checkRelFns = append(checkRelFns, fn)
	}

	structInfo := StructInfo{
		Fields:           []FieldInfo{},
		Relations:        []RelationInfo{},
		UpdateConditions: []model.UpdateConditions{},
	}

	for _, field := range schem.Fields {
		if field.DBName != "" && checkFn(checkFieldFns, field) {
			structInfo.Fields = append(structInfo.Fields, GetFieldInfo(c, field))
		}
		if _, ok := field.Tag.Lookup("query"); ok {
			fieldInfo := GetFieldInfo(c, field)
			fieldInfo.Query = true
			structInfo.Fields = append(structInfo.Fields, fieldInfo)
		}
	}

	for key, rel := range schem.Relationships.Relations {
		if !strings.HasPrefix(key, "_") && checkFn(checkRelFns, rel.Field) {
			structInfo.Relations = append(structInfo.Relations, GetRelationInfo(c, rel, relations))
		}
	}

	sort.SliceStable(structInfo.Relations, func(i, j int) bool {
		return structInfo.Relations[i].Label < structInfo.Relations[j].Label
	})

	mdl := reflect.New(schem.ModelType).Interface()
	if updateModel, ok := mdl.(model.UpdateConditionsModel); ok {
		structInfo.UpdateConditions = updateModel.UpdateConditions()
	}
	return structInfo
}

func GetFieldInfo(c *gin.Context, field *schema.Field) FieldInfo {
	requiredWithout := ""
	requiredWith := ""
	v := strings.Split(field.Tag.Get("valid"), ",")
	for _, req := range v {
		if strings.HasPrefix(req, "required_without=") {
			requiredWithout = req[17:]
		}
		if strings.HasPrefix(req, "required_with=") {
			requiredWith = req[14:]
		}
	}

	_, isDeprecated := field.Tag.Lookup("deprecated")

	fieldInfo := FieldInfo{
		Field:           field.Name,
		Label:           FieldToString(c, field),
		Descriptive:     field.Tag.Get("desc"),
		Type:            strings.ReplaceAll(field.StructField.Type.String(), "*", ""),
		Primary:         strings.Contains(field.Tag.Get("gorm"), "primaryKey"),
		UpdateKey:       strings.Contains(field.Tag.Get("import"), "updateKey"),
		Deprecated:      isDeprecated,
		RequiredWithout: requiredWithout,
		RequiredWith:    requiredWith,
		Updatable:       field.Updatable,
		Creatable:       field.Creatable,
	}

	label := field.Tag.Get("label")
	if label != "" {
		fieldInfo.Label = label
	}

	validationsTag := field.Tag.Get("validate")
	if len(validationsTag) > 0 {
		validations := strings.Split(validationsTag, ",")
		for _, validation := range validations {
			if validation == "required" {
				fieldInfo.Required = true
			} else if strings.Contains(validation, "max=") {
				val, err := strconv.Atoi(strings.ReplaceAll(validation, "max=", ""))
				if err == nil {
					fieldInfo.MaxLength = val
				}
			}
		}
	}
	return fieldInfo
}

func GetRelationInfo(c *gin.Context, rel *schema.Relationship, relations [][]string) RelationInfo {
	gormTags := strings.Split(rel.Field.Tag.Get("gorm"), ";")
	relationInfo := RelationInfo{
		Field: rel.Field.Name,
		Label: FieldToString(c, rel.Field),
	}

	typ := rel.Field.StructField.Type

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}

	ctrl := FindControllerByModel(typ)
	if ctrl != nil {
		relationInfo.Endpoint = ctrl.GetEndpointPath()
	}

	for _, tag := range gormTags {
		if strings.Contains(tag, "foreignKey:") {
			relationInfo.ForeignKey = strings.ReplaceAll(tag, "foreignKey:", "")
		} else if strings.Contains(tag, "references:") {
			relationInfo.References = strings.ReplaceAll(tag, "references:", "")
		}
		relationInfo.Updatable = rel.Field.Updatable
		relationInfo.Creatable = rel.Field.Creatable
	}

	relationInfo.Type = string(rel.Type)
	if relationInfo.Type == "has_one" {
		fld := rel.Schema.LookUpField(relationInfo.References)
		if fld != nil && !fld.PrimaryKey {
			relationInfo.Type = "belongs_to"
		}
	}
	rels := [][]string{}
	for _, relation := range relations {
		if len(relation) > 0 && relation[0] == relationInfo.Field {
			rels = append(rels, relation[1:])
		}
	}
	if len(rels) > 0 {
		structInfo := GetStructInfo(c, rel.FieldSchema, rels)
		relationInfo.Struct = &structInfo
	}

	return relationInfo
}

func FieldToString(c *gin.Context, field *schema.Field) string {
	return SentenceCase(strings.ReplaceAll(field.Name, "_", " "))
}
