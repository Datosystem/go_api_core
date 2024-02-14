package params

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Datosystem/go_api_core/message"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/schema"
)

func parseParams(c *gin.Context, modelSchema *schema.Schema, alias string, params []interface{}, conds *Conditions, allowed map[string]struct{}) message.Message {
	var operator string
	for _, item := range params {
		switch v := item.(type) {
		case string:
			if item == "OR" {
				operator = " OR "
			}
		case map[string]interface{}:
			if rawField, ok := v["field"]; ok {
				// Structured parameter
				var field string
				if field, ok = rawField.(string); ok {
					_, ok := allowed[field]
					if allowed == nil || ok {
						if parsedField, args, found := parseField(c, modelSchema, alias, field, conds.Nested); found {
							conds.Args = append(conds.Args, args...)
							err := parseStructuredParam(c, parsedField, v, operator, conds)
							if err != nil {
								return err
							}
						} else {
							return message.InvalidField(c, field)
						}
					}
				} else {
					return message.InvalidParamType(c, "field", "string")
				}
			} else {
				// Field parameter: value
				if len(v) > 1 {
					addOperator(&operator, &(*conds).Query)
					conds.Query += "("
				}
				for field, value := range v {
					_, ok := allowed[field]
					if allowed == nil || ok {
						if parsedField, args, found := parseField(c, modelSchema, alias, field, conds.Nested); found {
							conds.Args = append(conds.Args, args...)

							parseDynamicParam(parsedField, value, operator, conds)
						} else {
							return message.InvalidField(c, field)
						}
					}
				}
				if len(v) > 1 {
					conds.Query += ")"
				}
			}
		case []interface{}:
			addOperator(&operator, &(*conds).Query)
			conds.Query += "("
			err := parseParams(c, modelSchema, alias, v, conds, allowed)
			if err != nil {
				return err
			}
			conds.Query += ")\n"
		default:
			return message.Conflict(c).Text(fmt.Sprintf("invalid parameter of type %T", v))
		}
	}
	return nil
}

func parseField(c *gin.Context, modelSchema *schema.Schema, alias, key string, relations map[string]*Conditions) (string, []any, bool) {
	var field *schema.Field
	var table string
	if strings.Contains(key, ".") {
		pieces := strings.Split(key, ".")
		for i := 0; i < len(pieces)-1; i++ {
			rel, ok := modelSchema.Relationships.Relations[pieces[i]]
			if ok {
				modelSchema = rel.FieldSchema
			} else {
				return "", []any{}, false
			}
		}
		field = modelSchema.LookUpField(pieces[len(pieces)-1])
		table = strings.Join(pieces[:len(pieces)-1], "__")
		relations[strings.Join(pieces[:len(pieces)-1], ".")] = &Conditions{}
	} else {
		field = modelSchema.LookUpField(key)
		if alias != "" {
			table = alias
		} else {
			table = modelSchema.Table
		}
	}
	if field == nil {
		return key, []any{}, false
	} else if alias, ok := field.Tag.Lookup("alias"); ok {
		return alias, []any{}, true
	} else if funcName, ok := field.StructField.Tag.Lookup("query"); ok {
		if len(funcName) == 0 {
			funcName = "Query" + field.Name
		}
		mdl := reflect.New(modelSchema.ModelType)
		fn := mdl.MethodByName(funcName)
		if fn.IsValid() {
			var sel string
			args := []any{}
			rels := map[string]*Conditions{}
			msg := fn.Interface().(func(*gin.Context, any, *schema.Schema, string, bool, *string, *[]any, map[string]*Conditions) message.Message)(c, mdl.Interface(), modelSchema, table, false, &sel, &args, rels)
			if msg != nil {
				panic(msg)
			}
			for rel := range rels {
				relations[rel] = &Conditions{}
			}
			return sel, args, true
		} else {
			return "", []any{}, false
		}
	} else {
		return table + "." + field.DBName, []any{}, true
	}
}

func parseStructuredParam(c *gin.Context, field string, param map[string]interface{}, operator string, conds *Conditions) message.Message {
	addOperator(&operator, &(*conds).Query)
	op, _ := param["operator"].(string)
	conditionOperator := strings.ToUpper(strings.TrimSpace(op))
	if len(conditionOperator) == 0 {
		if val2, _ := param["value2"].(string); len(val2) > 0 {
			conditionOperator = "BETWEEN"
		} else {
			switch param["value"].(type) {
			case []interface{}:
				conditionOperator = "IN"
			case string:
				conditionOperator = "LIKE"
			default:
				conditionOperator = "="
			}
		}
	}
	switch conditionOperator {
	case "=", "!=", "<>", ">", "<", ">=", "<=", "LIKE":
		conds.Query += field + " " + conditionOperator + " ?"
		conds.Args = append(conds.Args, param["value"])
	case "BETWEEN":
		conds.Args = append(conds.Args, param["value"])
		conds.Args = append(conds.Args, param["value2"])
		conds.Query += field + " " + conditionOperator + " ? AND ?"
	case "IN", "NOT IN":
		conds.Query += field + " " + conditionOperator + " (?)"
		conds.Args = append(conds.Args, param["value"])
	case "IS NULL", "IS NOT NULL":
		conds.Query += field + " " + conditionOperator
	default:
		return message.InvalidParamOperator(c, conditionOperator)
	}
	return nil
}

func parseDynamicParam(field string, value interface{}, operator string, conds *Conditions) {
	addOperator(&operator, &(*conds).Query)
	var conditionOperator string
	switch value.(type) {
	case []interface{}:
		conditionOperator = "IN"
	default:
		conditionOperator = "="
	}
	conds.Query += field + " " + conditionOperator + " ?"
	conds.Args = append(conds.Args, value)
}

func addOperator(operator *string, stmt *string) {
	if len(*stmt) > 0 && (*stmt)[len(*stmt)-1:] != "(" {
		if len(*operator) == 0 {
			*stmt += " AND "
		} else {
			*stmt += *operator
		}
		*operator = ""
	}
}
