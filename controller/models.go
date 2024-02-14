package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/Datosystem/go_api_core/message"
	"github.com/Datosystem/go_api_core/model"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

type ErrorValidation struct {
	message.Msg
	Field     string
	Validator string
	Value     interface{}
}

func GetPrimaryName(model interface{}) (string, error) {
	val := reflect.Indirect(reflect.ValueOf(model))
	typ := val.Type()
	num := typ.NumField()
	for i := 0; i < num; i++ {
		field := typ.Field(i)
		if field.Anonymous && field.Name != "BaseModel" && field.Name != "Timestamps" {
			if key, err := GetPrimaryName(val.Field(i).Interface()); err == nil {
				return key, nil
			}
		}
		tag := field.Tag.Get("gorm")
		if strings.Contains(tag, "primaryKey") {
			return field.Name, nil
		}
	}
	return "", errors.New("primary key not found")
}

func GetPrimary(model interface{}) (int, error) {
	values := reflect.Indirect(reflect.ValueOf(model))
	key, err := GetPrimaryName(model)
	if err != nil {
		return 0, err
	} else {
		return int(values.FieldByName(key).Int()), nil
	}
}

func SetPrimary(model interface{}, value int) error {
	key, err := GetPrimaryName(model)
	if err != nil {
		return err
	}
	val := reflect.Indirect(reflect.ValueOf(model))
	val.FieldByName(key).SetInt(int64(value))
	return nil
}

func GetPrimaryFields(modelType reflect.Type) []string {
	primaryFields := []string{}
	numFields := modelType.NumField()
	for i := 0; i < numFields; i++ {
		fieldStruct := modelType.Field(i)
		if fieldStruct.Anonymous {
			primaryFields = append(primaryFields, GetPrimaryFields(fieldStruct.Type)...)
		} else {
			tagSettings := schema.ParseTagSetting(fieldStruct.Tag.Get("gorm"), ";")
			if val, ok := tagSettings["PRIMARYKEY"]; ok && utils.CheckTruth(val) {
				primaryFields = append(primaryFields, fieldStruct.Name)
			}
		}
	}
	return primaryFields
}

func GetPathParams(c *gin.Context, model interface{}, fields []string, destination interface{}) {
	if c.IsAborted() {
		return
	}
	dest := reflect.ValueOf(destination).Elem()
	mdl := reflect.ValueOf(model).Elem()
	for _, field := range fields {
		_, found := mdl.Type().FieldByName(field)
		val := c.Param(field)
		if len(val) == 0 || !found {
			message.InvalidUrlParameter(c, field).Abort(c)
			return
		}

		msg := assignValue(c, mdl.FieldByName(field), field, val, dest)
		if msg != nil {
			msg.Abort(c)
			return
		}
	}
}

func PathParamsToModels(c *gin.Context, modelType reflect.Type, fields []string, destination *[]interface{}) {
	if c.IsAborted() {
		return
	}
	var values = make([][]string, len(fields))
	for i, field := range fields {
		_, found := modelType.FieldByName(field)
		val := c.Param(field)
		values[i] = strings.Split(val, ",")
		if len(val) == 0 || !found || (i > 0 && len(values[i]) != len(values[i-1])) {
			message.InvalidUrlParameter(c, field).Abort(c)
			return
		}
	}

	for i := 0; i < len(values[0]); i++ {
		item := reflect.New(modelType).Elem()
		for j, field := range fields {
			msg := assignValue(c, item.FieldByName(field), field, values[j][i], item)
			if msg != nil {
				msg.Abort(c)
				return
			}
		}
		*destination = append(*destination, item.Addr().Interface())
	}
}

func assignValue(c *gin.Context, mdlField reflect.Value, field, val string, dest reflect.Value) message.Message {
	switch mdlField.Interface().(type) {
	case int:
		{
			intVal, err := strconv.Atoi(val)
			if err != nil {
				return message.InvalidParamType(c, field, "int")
			}
			applyValue(c, dest, field, intVal)
		}
	case int16:
		{
			intVal, err := strconv.Atoi(val)
			if err != nil {
				return message.InvalidParamType(c, field, "int")
			}
			applyValue(c, dest, field, int16(intVal))
		}
	case int32:
		{
			intVal, err := strconv.Atoi(val)
			if err != nil {
				return message.InvalidParamType(c, field, "int")
			}
			applyValue(c, dest, field, int32(intVal))
		}
	case int64:
		{
			intVal, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return message.InvalidParamType(c, field, "int")
			}
			applyValue(c, dest, field, intVal)
		}
	case *int:
		{
			intVal, err := strconv.Atoi(val)
			if err != nil {
				return message.InvalidParamType(c, field, "int")
			}
			applyValue(c, dest, field, &intVal)
		}
	case *int16:
		{
			intVal, err := strconv.Atoi(val)
			if err != nil {
				return message.InvalidParamType(c, field, "int")
			}
			i := int16(intVal)
			applyValue(c, dest, field, &i)
		}
	case *int32:
		{
			intVal, err := strconv.Atoi(val)
			if err != nil {
				return message.InvalidParamType(c, field, "int")
			}
			i := int32(intVal)
			applyValue(c, dest, field, &i)
		}
	case *int64:
		{
			intVal, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return message.InvalidParamType(c, field, "int")
			}
			applyValue(c, dest, field, &intVal)
		}
	case string:
		{
			applyValue(c, dest, field, val)
		}
	default:
		{
			return message.UnsupportedParamType(c, mdlField.Type().Name())
		}
	}
	return nil
}

func applyValue(c *gin.Context, destination reflect.Value, field string, value interface{}) {
	switch destination.Type().Kind() {
	case reflect.Map:
		{
			destination.SetMapIndex(reflect.ValueOf(field), reflect.ValueOf(value))
		}
	case reflect.Struct:
		{
			destination.FieldByName(field).Set(reflect.ValueOf(value))
		}
	default:
		{
			message.InternalServerError(c).Abort(c)
		}
	}
}

func LoadModel(c *gin.Context, jsonData []byte, model interface{}) {
	if c.IsAborted() {
		return
	}

	err := json.Unmarshal(jsonData, model)

	if err != nil {
		message.InvalidJSON(c).Text(err.Error()).Abort(c)
	}
}

func ValidateStruct(c *gin.Context, mdl interface{}) error {
	if validationModel, ok := mdl.(model.ValidationModel); ok {
		if msg := validationModel.Validate(c); msg != nil {
			return msg
		}
	}
	validate := validator.New()
	err := validate.Struct(mdl)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			return nil
		}

		campiObbligatori := ""
		for _, err := range err.(validator.ValidationErrors) {
			campiObbligatori += err.Field() + " " + err.ActualTag() + " " + err.Param()
		}

		return errors.New(campiObbligatori)
	}
	return nil
}

func ValidateModel(c *gin.Context, model interface{}) {
	if c.IsAborted() {
		return
	}

	err := ValidateStruct(c, model)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, err.Error())
	}
}

func ValidateModels(c *gin.Context, models interface{}) {
	if c.IsAborted() {
		return
	}

	modelsSlice := reflect.Indirect(reflect.ValueOf(models))

	typ := modelsSlice.Type()
	if typ.Kind() != reflect.Slice {
		message.ExpectedSlice(c).Abort(c)
	}
	res := ""
	for i := 0; i < modelsSlice.Len(); i++ {
		err := ValidateStruct(c, modelsSlice.Index(i).Interface())
		if err != nil {
			res += "Riga " + strconv.Itoa(i) + ": " + err.Error() + "\n"
		}
	}

	if len(res) > 0 {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, res)
	}
}

func validateVar(c *gin.Context, value interface{}, rules string, field string) message.Message {
	validate := validator.New()
	err := validate.Var(value, rules)
	if err != nil {
		return message.InvalidFieldValue(c, field, rules, value)
	}
	return nil
}

func ValidateMap(c *gin.Context, jsonMap map[string]interface{}, modelType reflect.Type) []error {
	var errors []error
	for key, value := range jsonMap {
		field, found := modelType.FieldByName(key)
		if found {
			rules := field.Tag.Get("validate")
			msg := validateVar(c, value, rules, field.Name)
			if msg != nil {
				errors = append(errors, msg)
			}
		} else {
			delete(jsonMap, key)
		}
	}
	return errors
}

func ValidateMaps(c *gin.Context, jsonMaps []map[string]interface{}, modelType reflect.Type) error {
	var err string
	for i, jsonMap := range jsonMaps {
		var errString string
		errors := ValidateMap(c, jsonMap, modelType)
		for _, err := range errors {
			errString += err.Error() + "\n"
		}

		if errString != "" {
			err += "Riga " + strconv.Itoa(i) + ":\n" + errString
		}
	}
	if err == "" {
		return nil
	}
	return fmt.Errorf(err)
}

func LoadAndValidateMap(c *gin.Context, jsonData []byte, jsonMap map[string]interface{}, modelType reflect.Type) {
	if c.IsAborted() {
		return
	}

	LoadModel(c, jsonData, &jsonMap)
	if c.IsAborted() {
		return
	}

	errors := ValidateMap(c, jsonMap, modelType)
	var errString string
	for _, err := range errors {
		errString += err.Error()
	}

	if len(jsonMap) == 0 {
		message.Unprocessable(c).Abort(c)
		return
	}

	if len(errors) > 0 {
		message.Conflict(c).Text(errString).Abort(c)
	}
}

func LoadAndValidateMaps(c *gin.Context, jsonData []byte, jsonMaps *[]map[string]interface{}, modelType reflect.Type) {
	if c.IsAborted() {
		return
	}

	LoadModel(c, jsonData, jsonMaps)
	if c.IsAborted() {
		return
	}

	err := ValidateMaps(c, *jsonMaps, modelType)

	if len(*jsonMaps) == 0 {
		message.Unprocessable(c).Abort(c)
		return
	}

	if err != nil {
		message.Conflict(c).Text(err.Error()).Abort(c)
	}
}

func ValidateMapsPrimaries(c *gin.Context, jsonMaps []map[string]interface{}, primaryKeys []string) {
	if c.IsAborted() {
		return
	}

	var err string
	for i, jsonMap := range jsonMaps {
		var errString string
		for _, field := range primaryKeys {
			if jsonMap[field] == nil {
				errString += message.InvalidFieldRequired(c, field).Error() + "\n"
			}
		}
		if errString != "" {
			err += message.RowError(c, i, "\n"+errString).Error()
		}
	}

	if err != "" {
		message.Conflict(c).Text(err).Abort(c)
	}
}

func GetMapKeys(mapToFlatten map[string]interface{}) []string {
	var keys []string
	for k := range mapToFlatten {
		keys = append(keys, k)
	}
	return keys
}
