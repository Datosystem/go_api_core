package model

import (
	"sort"
	"strings"

	"github.com/Datosystem/go_api_core/message"
	"github.com/Datosystem/go_api_core/params"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type UpdateConditions struct {
	Name       string                 `json:"name"`
	Default    bool                   `json:"default"`
	Conditions map[string]interface{} `json:"conditions"`
}

type UpdateCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type ConditionsModel interface {
	DefaultConditions(*gorm.DB, string) (query string, args []interface{})
}

type UpdateConditionsModel interface {
	UpdateConditions() []UpdateConditions
}

type JoinsModel interface {
	DefaultJoins(*gorm.DB, string) string
}

type OrderedModel interface {
	DefaultOrder(*gorm.DB, string) string
}

type ValidationModel interface {
	Validate(*gin.Context) message.Message
}

type TableModel interface {
	TableName() string
}

type DisplayNameRelationsModel interface {
	DisplayNameRelations() []string
}

type DisplayNamePatternModel interface {
	DisplayNamePattern() string
}

type tableField struct {
	Table string
	Field *schema.Field
}

type BaseModel struct {
	Delete       bool   `gorm:"-" json:"$delete,omitempty"`
	DISPLAY_NAME string `gorm:"-" query:"" json:",omitempty" label:"Nome di visualizzazione"`
}

func (BaseModel) QueryDISPLAY_NAME(c *gin.Context, model interface{}, modelSchema *schema.Schema, table string, nested bool, query *string, args *[]any, rels map[string]*params.Conditions) message.Message {
	if m, ok := model.(DisplayNamePatternModel); ok {
		pattern := m.DisplayNamePattern()
		sel, relSet := DisplayPatternToSql(pattern, modelSchema, table, nested)
		for rel := range relSet {
			rels[rel] = &params.Conditions{}
		}
		*query = "LTRIM(RTRIM(" + sel + "))"
		return nil
	} else {
		if m, ok := model.(DisplayNameRelationsModel); ok {
			for _, rel := range m.DisplayNameRelations() {
				if nested {
					rels[strings.ReplaceAll(table, "__", ".")+"."+rel] = &params.Conditions{}
				} else {
					rels[rel] = &params.Conditions{}
				}
			}
		}

		fields := []tableField{}
		for rel := range rels {
			relSchema := modelSchema
			pieces := strings.Split(rel, ".")
			for _, piece := range pieces {
				if r, ok := relSchema.Relationships.Relations[piece]; ok {
					relSchema = r.FieldSchema
				} else {
					panic("Invalid relation " + rel)
				}
			}
			alias := strings.ReplaceAll(rel, ".", "__")
			relFields := []tableField{}
			for _, field := range relSchema.Fields {
				if strings.HasPrefix(field.Tag.Get("desc"), "1.") {
					relFields = append(relFields, tableField{Table: alias, Field: field})
				}
			}

			sort.SliceStable(relFields, func(i, j int) bool {
				return relFields[i].Field.Tag.Get("desc") < relFields[j].Field.Tag.Get("desc")
			})

			fields = append(fields, relFields...)
		}
		relFields := []tableField{}
		for _, field := range modelSchema.Fields {
			if strings.HasPrefix(field.Tag.Get("desc"), "1.") {
				relFields = append(relFields, tableField{Table: table, Field: field})
			}
		}

		sort.SliceStable(relFields, func(i, j int) bool {
			return relFields[i].Field.Tag.Get("desc") < relFields[j].Field.Tag.Get("desc")
		})

		fields = append(fields, relFields...)

		if len(fields) == 0 {
			return message.DisplayNameNotSupported(c)
		}

		var sel string
		if len(fields) > 0 {
			t := fields[0].Table
			for i := range fields {
				if fields[i].Table != t {
					sel += "+ ' - ' +"
				}
				sel += DisplayFieldToSql(fields[i].Table, fields[i].Field, i > 0)
			}
		}
		*query = "LTRIM(RTRIM(" + sel + "))"
		return nil
	}
}

func DisplayFieldToSql(table string, field *schema.Field, concat bool) string {
	sel := "CASE WHEN " + table + "." + field.DBName + " IS NOT NULL"
	if field.DataType == schema.String {
		sel += " AND LEN(" + table + "." + field.DBName + ") > 0"
	}
	sel += " THEN"
	if concat {
		sel = " + " + sel + " ' ' + "
	}
	if field.DataType == schema.String {
		sel += table + "." + field.DBName
	} else {
		sel += " CAST(" + table + "." + field.DBName + " AS NVARCHAR(MAX))"
	}
	return sel + " ELSE '' END"
}

func DisplayPatternToSql(pattern string, modelSchema *schema.Schema, table string, nested bool) (string, map[string]*params.Conditions) {
	var startIndex int
	var sel string
	relSet := map[string]*params.Conditions{}
	for i, char := range pattern {
		if string(char) == "{" {
			if i-startIndex > 0 {
				if len(sel) > 0 {
					sel += "+"
				}
				sel += "'" + pattern[startIndex:i] + "'"
			}
			startIndex = i + 1
		} else if string(char) == "}" {
			pieces := strings.Split(pattern[startIndex:i], ".")
			t := table
			relSchema := modelSchema
			if len(pieces) > 1 {
				for _, piece := range pieces[:len(pieces)-1] {
					if r, ok := relSchema.Relationships.Relations[piece]; ok {
						relSchema = r.FieldSchema
					} else {
						panic("Invalid field " + pattern[startIndex:i])
					}
				}
				rel := strings.Join(pieces[:len(pieces)-1], ".")
				if nested {
					relSet[strings.ReplaceAll(table, "__", ".")+"."+rel] = &params.Conditions{}
					t = table + "__" + strings.ReplaceAll(rel, ".", "__")
				} else {
					relSet[rel] = &params.Conditions{}
					t = strings.ReplaceAll(rel, ".", "__")
				}
			}
			if len(sel) > 0 {
				sel += "+"
			}
			sel += DisplayFieldToSql(t, relSchema.LookUpField(pieces[len(pieces)-1]), false)
			startIndex = i + 1
		}
	}
	if len(pattern)-startIndex > 0 {
		if len(sel) > 0 {
			sel += "+"
		}
		sel += "'" + pattern[startIndex:] + "'"
	}
	return sel, relSet
}
