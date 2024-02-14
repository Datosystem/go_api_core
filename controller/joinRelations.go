package controller

import (
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/Datosystem/go_api_core/message"
	"github.com/Datosystem/go_api_core/model"
	"github.com/Datosystem/go_api_core/params"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const queryTableName = "ORIGIN"

type DynamicModel struct {
	Dest      interface{}
	Relations *map[string]NestedModel
}

type ModelInfo struct {
	Select     []string
	SelectArgs []any
	Fields     []reflect.StructField
	Schema     *schema.Schema
	Table      string
	Order      string
	Relations  map[string]*params.Conditions
	Nested     map[string]NestedModel
	Aggregate  bool
	Distinct   bool
}

type NestedModel struct {
	References []*schema.Reference
	Slice      bool
	ModelInfo  *ModelInfo
}

func (n NestedModel) NextNested() map[string]NestedModel {
	return n.ModelInfo.Nested
}

var fkAlias = "___FK___"

func JoinRelations(c *gin.Context, d *gorm.DB, config QueryMapConfig, modelInfo *ModelInfo, relations map[string]*params.Conditions) {
	joins := ""
	joinsArgs := []interface{}{}
	joinedTables := map[string]*schema.Schema{}

	// Check the relations

	relationsKeys := make([]string, 0, len(relations))
	for k := range relations {
		relationsKeys = append(relationsKeys, k)
	}
	sort.Strings(relationsKeys)

	for _, relation := range relationsKeys {
		conditions := relations[relation]
		relSchema := modelInfo.Schema
		pieces := strings.Split(relation, ".")
		for i, piece := range pieces {
			alias := strings.Join(pieces[:i+1], "__")
			if rel, ok := relSchema.Relationships.Relations[piece]; ok {
				relSchema = rel.FieldSchema
				if joinedTables[alias] == nil {
					joinedTables[alias] = rel.FieldSchema
					if conditions.Type != "I" {
						// Outer join
						joins += ` LEFT`
					}
					joins += ` JOIN ` + rel.FieldSchema.Table + ` AS ` + alias + ` ON`
					for j, ref := range rel.References {
						if j > 0 {
							joins += " AND "
						}
						if i > 0 {
							joins += ` ` + strings.Join(pieces[:i], "__") + `.` + ref.PrimaryKey.DBName + ` = ` + alias + `.` + ref.ForeignKey.DBName
						} else {
							joins += ` ` + modelInfo.Table + `.` + ref.PrimaryKey.DBName + ` = ` + alias + `.` + ref.ForeignKey.DBName
						}
					}
					if !config.SkipDefaults {
						mdl := reflect.New(joinedTables[alias].ModelType).Interface()
						if model, ok := mdl.(model.ConditionsModel); ok {
							query, args := model.DefaultConditions(d, alias)
							if query != "" {
								joins += " AND " + query
								joinsArgs = append(joinsArgs, args...)
							}
						}
						if model, ok := mdl.(model.JoinsModel); ok {
							joins += model.DefaultJoins(d, alias)
						}
					}
				}
			} else {
				d.AddError(message.InvalidRelations(c, strings.Join(pieces[:i+1], ".")))
				return
			}
		}
		if conditions.Query != "" && (conditions.Type == "I" || conditions.Type == "O") {
			joins += " AND" + conditions.Query
			joinsArgs = append(joinsArgs, conditions.Args...)
		}
	}

	if d.Error != nil {
		return
	}

	if len(joins) > 0 {
		d.Joins(joins, joinsArgs...)
	}
}

func ModelToTableNames(stmt *string, modelSchema *schema.Schema) []string {
	relations := []string{}
	r := regexp.MustCompile(`([\w\.]*)\.(\w*)`)
	r.ReplaceAllStringFunc(*stmt, func(s string) string {
		slice := strings.Split(s, ".")
		if slice[0] == modelSchema.Table {
			return s
		}
		relations = append(relations, strings.Join(slice[:len(slice)-1], "."))
		return strings.Join(slice[:len(slice)-1], "__") + "." + slice[len(slice)-1]
	})
	return relations
}

func GetModelInfo(c *gin.Context, modelSchema *schema.Schema, selects string, computedFields map[string]string, modelInfo *ModelInfo, args *QueryMapArgs) message.Message {
	modelInfo.Table = strings.TrimSpace(modelInfo.Schema.Table)
	if strings.HasSuffix(modelInfo.Table, ")") {
		modelInfo.Table = queryTableName
	} else if index := strings.LastIndex(modelInfo.Table, ") AS "); index != -1 {
		modelInfo.Table = modelInfo.Table[index+5:]
	}

	if len(selects) > 0 {
		if strings.HasPrefix(selects, "DISTINCT ") {
			selects = selects[9:]
			modelInfo.Distinct = true
		}

		fields := strings.Split(selects, ",")
		for j, field := range fields {
			field = strings.TrimSpace(field)
			if len(field) == 0 {
				continue
			}

			fieldAlias := ""
			index := strings.Index(field, " AS ")
			if index > -1 {
				fieldAlias = field[index+4:]
				field = field[:index]
			}

			sum := false
			count := false
			if strings.HasPrefix(field, "SUM(") && strings.HasSuffix(field, ")") {
				field = field[4 : len(field)-1]
				sum = true
			}
			if strings.HasPrefix(field, "COUNT(") && strings.HasSuffix(field, ")") {
				field = field[6 : len(field)-1]
				count = true
			}

			if pos := strings.Index(field, "<."); pos != -1 {
				if j > 0 {
					if index := strings.LastIndex(fields[j-1], "."); index != -1 {
						field = field[:pos] + fields[j-1][:index] + field[pos+1:]
					}
				}
				fields[j] = field
			}

			relSchema := modelSchema
			info := modelInfo
			pieces := strings.Split(field, ".")
			startIndex := 0
			key := ""
			for i := 0; i < len(pieces)-1; i++ {
				var nested bool
				if strings.HasPrefix(pieces[i], ">") {
					pieces[i] = pieces[i][1:]
					nested = true
				}
				if key != "" {
					key += "."
				}
				key += pieces[i]
				if rel, ok := relSchema.Relationships.Relations[pieces[i]]; ok {
					relSchema = rel.FieldSchema
					if nested {
						if _, ok := info.Nested[key]; !ok {
							n := NestedModel{
								References: rel.References,
								ModelInfo:  &ModelInfo{Fields: []reflect.StructField{}, Relations: map[string]*params.Conditions{}, Nested: map[string]NestedModel{}, Schema: relSchema},
							}
							n.ModelInfo.Table = n.ModelInfo.Schema.Table
							if strings.HasSuffix(strings.TrimSpace(n.ModelInfo.Schema.Table), ")") {
								n.ModelInfo.Table = queryTableName
							}
							mdl := reflect.New(relSchema.ModelType).Interface()
							if ordMdl, ok := mdl.(model.OrderedModel); ok {
								n.ModelInfo.Order = ordMdl.DefaultOrder(c.MustGet("db").(*gorm.DB), n.ModelInfo.Table)
							}
							l := len(rel.References)
							var fk string
							for j, ref := range rel.References {
								if j > 0 {
									fk += ",'___',"
								}
								if l > 1 {
									fk += "CAST(" + n.ModelInfo.Table + "." + ref.ForeignKey.DBName + " AS NVARCHAR(MAX))"
								} else {
									fk += n.ModelInfo.Table + "." + ref.ForeignKey.DBName
								}
							}
							if len(rel.References) > 1 {
								fk = "CONCAT(" + fk + ")"
							}
							n.ModelInfo.Select = []string{fk + " AS " + fkAlias}
							if rel.Field.FieldType.Kind() == reflect.Slice {
								n.Slice = true
							}
							info.Nested[key] = n
						}
						startIndex = i + 1
						info = info.Nested[key].ModelInfo
						key = ""
					}
				} else {
					return message.InvalidRelation(c, strings.Join(pieces[:i+1], "."))
				}
			}
			fieldName := pieces[len(pieces)-1]
			if fieldAlias != "" {
				fieldAlias = strings.ReplaceAll(fieldAlias, "*", fieldName)
			}

			var structFields []*schema.Field

			if strings.HasSuffix(fieldName, "*") {
				structFields = []*schema.Field{}
				for _, fld := range relSchema.Fields {
					_, okQ := fld.Tag.Lookup("query")
					_, okC := fld.Tag.Lookup("compute")
					if !okQ && !okC {
						structFields = append(structFields, fld)
					}
				}
			} else {
				fld := relSchema.LookUpField(fieldName)
				if fld == nil {
					return message.InvalidField(c, field)
				}
				if (sum || count) && fieldAlias == "" {
					fieldAlias = fieldName
				}
				structFields = []*schema.Field{fld}
			}

			if len(pieces)-startIndex > 1 {
				info.Relations[strings.Join(pieces[startIndex:len(pieces)-1], ".")] = &params.Conditions{}
			}

			for _, field := range structFields {
				var sel string
				table := info.Table
				if len(pieces)-startIndex > 1 {
					table = strings.Join(pieces[startIndex:len(pieces)-1], "__")
				}
				if funcName, ok := field.StructField.Tag.Lookup("compute"); ok {
					if len(funcName) == 0 {
						funcName = "Compute" + field.Name
					}
					computedFields[strings.Join(pieces, ".")] = funcName
					// Here, instead of funcName, I would include the entire function
				} else if funcName, ok := field.StructField.Tag.Lookup("query"); ok {
					field.StructField.Tag = reflect.StructTag(strings.TrimPrefix(string(field.StructField.Tag), `gorm:"-"`))
					if len(funcName) == 0 {
						funcName = "Query" + field.Name
					}
					mdl := reflect.New(relSchema.ModelType)
					fn := mdl.MethodByName(funcName)
					if fn.IsValid() {
						rels := map[string]*params.Conditions{}
						msg := fn.Interface().(func(*gin.Context, any, *schema.Schema, string, bool, *string, *[]any, map[string]*params.Conditions) message.Message)(c, mdl.Interface(), relSchema, table, len(pieces)-startIndex > 1, &sel, &info.SelectArgs, rels)
						if msg != nil {
							return msg
						}
						for rel := range rels {
							info.Relations[rel] = &params.Conditions{}
						}
						if len(fieldAlias) == 0 {
							fieldAlias = field.Name
						}
					} else {
						return message.InvalidField(c, field.Name)
					}
				} else if len(field.DBName) != 0 {
					sel = table + ".[" + field.DBName + "]"
				}
				if len(sel) > 0 {
					if sum {
						sel = "SUM(" + sel + ")"
						info.Aggregate = true
					}
					if count {
						sel = "COUNT(" + sel + ")"
						info.Aggregate = true
					}
					structField := field.StructField
					if len(fieldAlias) > 0 {
						r := regexp.MustCompile(`^[\w*]+$`)
						if !r.MatchString(fieldAlias) {
							return message.InvalidFieldAlias(c, fieldAlias, fieldName)
						}
						structField.Name = strings.ReplaceAll(fieldAlias, "*", field.Name)
						sel += " AS [" + structField.Name + "]"
					}
					info.Fields = append(info.Fields, structField)
					info.Select = append(info.Select, sel)
				}
			}
		}
	} else {
		for _, field := range modelSchema.Fields {
			if field.Readable && len(field.DBName) != 0 {
				modelInfo.Fields = append(modelInfo.Fields, field.StructField)
				modelInfo.Select = append(modelInfo.Select, modelInfo.Table+"."+field.DBName)
			}
		}
	}
	if len(args.Ord) > 0 {
		msg := ParseOrder(c, args.Ord, modelInfo)
		if msg != nil {
			return msg
		}
	} else if ordModel, ok := reflect.New(modelSchema.ModelType).Interface().(model.OrderedModel); ok {
		modelInfo.Order = ordModel.DefaultOrder(c.MustGet("db").(*gorm.DB), modelInfo.Table)
	} else if modelSchema.PrioritizedPrimaryField != nil {
		modelInfo.Order = modelInfo.Table + "." + modelSchema.PrioritizedPrimaryField.DBName
	}
	return nil
}

func ParseOrder(c *gin.Context, order string, info *ModelInfo) message.Message {
	if len(order) > 0 {
		local := []string{}
		nested := map[string][]string{}
		fields := strings.Split(order, ",")
		for i, field := range fields {
			field = strings.TrimSpace(field)
			if strings.HasPrefix(field, ">") {
				field = field[1:]
				pieces := strings.Split(field, ".")
				if _, ok := nested[pieces[0]]; !ok {
					nested[pieces[0]] = []string{}
				}
				nested[pieces[0]] = append(nested[pieces[0]], strings.Join(pieces[1:], "."))
			} else {
				var toBoolean bool
				for strings.HasPrefix(field, "!") {
					toBoolean = true
					field = field[1:]
				}

				if pos := strings.Index(field, "<."); pos != -1 {
					if i > 0 {
						if index := strings.LastIndex(fields[i-1], "."); index != -1 {
							field = field[:pos] + fields[i-1][:index] + field[pos+1:]
						}
					}
				}

				relSchema := info.Schema
				pieces := strings.Split(field, ".")
				for i := 0; i < len(pieces)-1; i++ {
					if rel, ok := relSchema.Relationships.Relations[strings.TrimPrefix(pieces[i], ">")]; ok {
						relSchema = rel.FieldSchema
					} else {
						return message.InvalidRelation(c, strings.Join(pieces[:i+1], "."))
					}
				}

				fldName := strings.TrimSuffix(strings.TrimSuffix(field, " DESC"), " ASC")
				piece := strings.TrimSuffix(strings.TrimSuffix(pieces[len(pieces)-1], " DESC"), " ASC")
				fld := relSchema.LookUpField(piece)
				if fld == nil {
					search := "[" + piece + "]"
					found := false
					for _, f := range info.Select {
						if asIndex := strings.LastIndex(f, " AS "); asIndex != -1 {
							if f[asIndex+4:] == search || strings.HasSuffix(f[:asIndex], search) {
								found = true
								break
							}
						} else if strings.HasSuffix(f, search) {
							found = true
							break
						}
					}
					if found {
						if strings.HasSuffix(pieces[len(pieces)-1], " DESC") {
							search += " DESC"
						}
						local = append(local, search)
					} else {
						return message.InvalidField(c, field)
					}
				}

				if fld != nil {
					if info.Distinct {
						found := false
						selField := "[" + fld.Name + "]"
						selRel := selField
						if len(pieces) > 1 {
							selRel = strings.Join(pieces[:len(pieces)-1], "__") + "." + selField
						}
						for _, f := range info.Select {
							if asIndex := strings.LastIndex(f, " AS "); asIndex != -1 {
								if f[asIndex+4:] == selField || strings.HasSuffix(f[:asIndex], selRel) {
									found = true
									break
								}
							} else if strings.HasSuffix(f, selRel) {
								found = true
								break
							}
						}
						if !found {
							return message.ConflictingOrderByAndDistinct(c)
						}
					}

					if _, ok := fld.StructField.Tag.Lookup("query"); ok {
						field = pieces[len(pieces)-1]
					} else {
						alias := info.Table
						l := len(pieces)
						if l > 1 {
							alias = strings.Join(pieces[:l-1], "__")
						}
						field = alias + "." + pieces[l-1]
					}

					if toBoolean {
						field = "CASE WHEN " + fldName + " IS NULL THEN 0 ELSE 1 END" + field[len(fldName):]
					}
					local = append(local, field)
				}
			}
		}

		for key, nestedArray := range nested {
			if n, ok := info.Nested[key]; ok {
				msg := ParseOrder(c, strings.Join(nestedArray, ","), n.ModelInfo)
				if msg != nil {
					return msg
				}
			} else {
				return message.InvalidRelation(c, key)
			}
		}

		info.Order = strings.Join(local, ",")
	}
	return nil
}

func CreateModel(c *gin.Context, info *ModelInfo, slice bool) (reflect.Type, message.Message) {
	names := map[string]struct{}{}
	fields := []reflect.StructField{}
	fields = append(fields, info.Fields...)
	for i, field := range fields {
		if _, ok := names[field.Name]; ok {
			return nil, message.DuplicateStructField(c, field.Name)
		}
		names[field.Name] = struct{}{}
		if field.Type.Kind() != reflect.Ptr {
			fields[i].Type = reflect.PointerTo(fields[i].Type)
		}
	}
	if len(info.Nested) > 0 {
		for name, rel := range info.Nested {
			typ, err := CreateModel(c, rel.ModelInfo, rel.Slice)
			if err != nil {
				return nil, err
			}
			if index := strings.LastIndex(name, "."); index != -1 {
				name = name[index+1:]
			}
			fields = append(fields, reflect.StructField{
				Name: name,
				Type: typ,
				Tag:  `gorm:"-"`,
			})
			// TODO: Examine in detail the purpose of this section
			/*if _, ok := names[rel.ForeignKey]; !ok {
				fields = append(fields, reflect.StructField{
					Name: rel.ForeignKey,
					Type: info.Schema.LookUpField(rel.ForeignKey).FieldType,
					Tag:  `json:"-"`,
				})
				names[rel.ForeignKey] = struct{}{}
			}*/
		}
	}

	t := reflect.PointerTo(reflect.StructOf(fields))
	if slice {
		t = reflect.SliceOf(t)
	}
	return t, nil
}

func DynamicQuery(c *gin.Context, info *ModelInfo, conditions *params.Conditions) func(*gorm.DB) *gorm.DB {
	return func(d *gorm.DB) *gorm.DB {
		if info == nil {
			return d
		}

		d = d.Select(info.Select)
		d.Statement.Distinct = info.Distinct
		if info.Aggregate {
			for _, field := range info.Select {
				if !strings.HasPrefix(field, "SUM(") && !strings.HasPrefix(field, "COUNT(") {
					d = d.Group(field)
				}
			}
		}

		if d.Statement.Table == "" {
			table := info.Schema.Table
			if table != info.Table && !strings.Contains(table, ") AS ") {
				table += " AS " + info.Table
			}
			d.Table(table)
		}

		mdl := reflect.New(info.Schema.ModelType).Interface()
		// Handles default conditions
		if model, ok := mdl.(model.ConditionsModel); ok {
			query, args := model.DefaultConditions(d, info.Table)
			d.Where(query, args...)
		}

		if model, ok := mdl.(model.JoinsModel); ok {
			d.Joins(model.DefaultJoins(d, info.Table))
		}

		if len(conditions.Query) > 0 {
			d.Where(conditions.Query, conditions.Args...)
		}

		JoinRelations(c, d, QueryMapConfig{}, info, RelationsFromModelInfo(info, conditions.Nested))
		return d
	}
}

func RelationsFromModelInfo(mdl *ModelInfo, otherRelations map[string]*params.Conditions) map[string]*params.Conditions {
	set := map[string]*params.Conditions{}
	for rel := range mdl.Relations {
		set[rel] = &params.Conditions{}
	}
	for rel := range otherRelations {
		if otherRelations[rel].Type != "N" {
			set[rel] = otherRelations[rel]
		}
	}
	return set
}

func GetRelations(c *gin.Context) []string {
	var relations []string
	if len(c.Query("rel")) > 0 {
		relations = strings.Split(strings.ReplaceAll(c.Query("rel"), " ", ""), ",")
	}
	return relations
}
