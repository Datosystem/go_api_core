package params

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/Datosystem/go_api_core/message"
	"github.com/gin-gonic/gin"
	"github.com/iancoleman/orderedmap"
	"gorm.io/gorm/schema"
)

func parseParamsV2(c *gin.Context, modelSchema *schema.Schema, alias string, params *orderedmap.OrderedMap, conds *Conditions, allowed map[string]struct{}) message.Message {
	for _, key := range params.Keys() {
		value, _ := params.Get(key)
		logicOp := regexp.MustCompile(`^[|]+`).FindString(key)
		condtionOp := regexp.MustCompile(`[!><%\-=]+$`).FindString(key)
		if key[len(logicOp):][0] == '>' {
			dotIndex := strings.Index(key, ".")
			if dotIndex == -1 {
				return message.InvalidField(c, key)
			}
			remainder := logicOp + key[dotIndex+1:]
			key = key[1+len(logicOp) : dotIndex]
			rel, ok := modelSchema.Relationships.Relations[key]
			if !ok {
				continue
			}
			if _, ok := conds.Nested[key]; ok {
				if conds.Nested[key].Type != "N" {
					conds.Nested[key].Type = "M"
				}
			} else {
				conds.Nested[key] = &Conditions{Type: "N", Nested: map[string]*Conditions{}}
			}
			nested := orderedmap.New()
			nested.Set(remainder, value)
			if err := parseParamsV2(c, rel.FieldSchema, "", nested, conds.Nested[key], allowed); err != nil {
				return err
			}
		} else if v, ok := value.(orderedmap.OrderedMap); ok {
			if key[0] == '?' || key[0] == '&' {
				cond := &Conditions{Nested: map[string]*Conditions{}}
				cond.Type = "I" // Inner
				if key[0] == '?' {
					cond.Type = "O" // Outer
				}
				key = key[1:]
				conds.Nested[key] = cond
				if len(v.Keys()) > 0 {
					if err := parseParamsV2(c, modelSchema, alias, &v, conds.Nested[key], allowed); err != nil {
						return err
					}
				}
			} else if len(v.Keys()) > 0 {
				addLogicOperator(logicOp, &(*conds).Query)
				if strings.Contains(condtionOp, "!") {
					conds.Query += " NOT"
				}
				conds.Query += "(\n"
				if err := parseParamsV2(c, modelSchema, alias, &v, conds, allowed); err != nil {
					return err
				}
				conds.Query += "\n)"
			}
		} else {
			addLogicOperator(logicOp, &(*conds).Query)
			field := key[len(logicOp) : len(key)-len(condtionOp)]
			_, ok := allowed[field]
			if allowed == nil || ok {
				if conds.Nested == nil {
					conds.Nested = map[string]*Conditions{}
				}
				if err := addCondition(c, modelSchema, alias, condtionOp, field, value, conds, conds.Nested); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func addLogicOperator(ops string, stmt *string) {
	if len(*stmt) > 0 && !strings.HasSuffix(*stmt, "(\n") {
		if strings.Contains(ops, "|") {
			*stmt += " OR"
		} else {
			*stmt += " AND"
		}
	}
}

func addCondition(c *gin.Context, modelSchema *schema.Schema, alias, ops string, key string, value interface{}, conds *Conditions, relations map[string]*Conditions) message.Message {
	if field, args, typ := parseFieldV2(c, modelSchema, alias, key, relations); typ != nil {
		conds.Query += " " + field
		conds.Args = append(conds.Args, args...)
	} else {
		return message.InvalidField(c, key)
	}

	var operator string
	if strings.Contains(ops, "><") {
		operator = ""
		if strings.Contains(ops, "!") {
			operator += " NOT"
		}
		operator += " BETWEEN ? AND ?"
		if slice, ok := value.([]interface{}); ok {
			conds.Args = append(conds.Args, slice[0], slice[1])
		} else {
			return message.InvalidParamType(c, key, "[]string")
		}
	} else if value == "nil" || value == "null" || value == nil {
		operator = " IS"
		if strings.Contains(ops, "!") {
			operator += " NOT"
		}
		operator += " NULL"
	} else {
		if strings.Contains(ops, "%") {
			operator = " LIKE ?"
		} else if _, ok := value.([]interface{}); ok {
			operator = " IN (?)"
		}

		if len(operator) > 0 {
			if strings.Contains(ops, "!") {
				operator = " NOT" + operator
			}
		} else {
			if strings.Contains(ops, ">=") {
				if strings.Contains(ops, "!") {
					operator = " < ?"
				} else {
					operator = " >= ?"
				}
			} else if strings.Contains(ops, ">") {
				if strings.Contains(ops, "!") {
					operator = " <= ?"
				} else {
					operator = " > ?"
				}
			} else if strings.Contains(ops, "<=") {
				if strings.Contains(ops, "!") {
					operator = " > ?"
				} else {
					operator = " <= ?"
				}
			} else if strings.Contains(ops, "<") {
				if strings.Contains(ops, "!") {
					operator = " >= ?"
				} else {
					operator = " < ?"
				}
			} else {
				operator = " "
				if strings.Contains(ops, "!") {
					operator += "!"
				}
				operator += "= ?"
			}
		}

		conds.Args = append(conds.Args, value)
	}

	conds.Query += operator
	return nil
}

func parseFieldV2(c *gin.Context, modelSchema *schema.Schema, alias, key string, relations map[string]*Conditions) (string, []any, reflect.Type) {
	var field *schema.Field
	var table string
	if strings.Contains(key, ".") {
		pieces := strings.Split(key, ".")
		for i := 0; i < len(pieces)-1; i++ {
			rel, ok := modelSchema.Relationships.Relations[pieces[i]]
			if ok {
				modelSchema = rel.FieldSchema
			} else {
				return "", []any{}, nil
			}
		}
		field = modelSchema.LookUpField(pieces[len(pieces)-1])
		table = strings.Join(pieces[:len(pieces)-1], "__")
		rel := strings.Join(pieces[:len(pieces)-1], ".")
		if _, ok := relations[rel]; ok {
			if relations[rel].Type != "" {
				relations[rel].Type = "M"
			}
		} else {
			relations[rel] = &Conditions{}
		}
	} else {
		field = modelSchema.LookUpField(key)
		if alias != "" {
			table = alias
		} else {
			table = modelSchema.Table
		}
	}
	if field == nil {
		return key, []any{}, nil
	} else if alias, ok := field.Tag.Lookup("alias"); ok {
		return alias, []any{}, field.FieldType
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
				if _, ok := relations[rel]; !ok {
					relations[rel] = &Conditions{}
				}
			}
			return sel, args, field.FieldType
		} else {
			return "", []any{}, nil
		}
	} else {
		return table + "." + field.DBName, []any{}, field.FieldType
	}
}
