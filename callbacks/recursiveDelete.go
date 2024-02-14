package callbacks

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

func RegisterRecursiveDelete(db *gorm.DB, prefix string) {
	db.Callback().Delete().Before("gorm:commit_or_rollback_transaction").Register(prefix+":recursive_delete", RecursiveDeleteCallback)
}

func RecursiveDeleteCallback(db *gorm.DB) {
	if db.Error != nil || db.Statement.Model == nil {
		return
	}

	val := db.Statement.ReflectValue

	relations := []*schema.Relationship{}
	for _, rel := range db.Statement.Schema.Relationships.HasOne {
		fld := rel.References[0].ForeignKey
		if fld != nil && !fld.PrimaryKey {
			relations = append(relations, rel)
		}
	}
	relations = append(relations, db.Statement.Schema.Relationships.HasMany...)

RelLoop:
	for _, rel := range relations {
		if !strings.HasPrefix(rel.FieldSchema.Table, "(") && rel.Field.Updatable {
			if len(rel.FieldSchema.PrimaryFieldDBNames) == 0 {
				continue
			}
			dest := reflect.New(reflect.SliceOf(rel.FieldSchema.ModelType)).Interface()
			keySet := map[string]struct{}{}
			for _, key := range rel.FieldSchema.PrimaryFieldDBNames {
				keySet[key] = struct{}{}
			}
			for key, nestedRel := range rel.FieldSchema.Relationships.Relations {
				if nestedRel.Field.Updatable && !strings.HasPrefix(key, "_") {
					for _, ref := range nestedRel.References {
						fieldName := ref.PrimaryKey.Name
						if !ref.OwnPrimaryKey {
							fieldName = ref.ForeignKey.Name
						}
						keySet[fieldName] = struct{}{}
					}
				}
			}
			keyArr := []string{}
			for key := range keySet {
				keyArr = append(keyArr, key)
			}
			tx := db.Session(&gorm.Session{NewDB: true}).Select(keyArr).Table(rel.FieldSchema.Table)
			exprs := []clause.Expression{}
			conditions := rel.ToQueryConditions(db.Statement.Context, val)
			for _, cond := range conditions {
				if in, ok := cond.(clause.IN); ok {
					if cols, ok := in.Column.([]clause.Column); ok {
						if len(in.Values) == 0 {
							return
						}
						if len(cols) > 1 {
							keySet := map[string]any{}
							var ks any = keySet
							vals := in.Values[0].([]any)
							for i, val := range vals {
								if reflect.TypeOf(val).Kind() == reflect.Ptr {
									if reflect.ValueOf(val).IsNil() {
										// If an invalid value is found skip to the next relation
										continue RelLoop
									}
									val = reflect.ValueOf(val).Elem().Interface()
								}
								str := fmt.Sprint(val)
								if i == len(vals)-1 {
									current := ks.(map[string]struct{})
									current[str] = struct{}{}
								} else if i == len(vals)-2 {
									current := ks.(map[string]any)
									if _, ok := current[str]; !ok {
										m := map[string]struct{}{}
										ks.(map[string]any)[str] = m
									}
									ks = current[str]
								} else {
									current := ks.(map[string]any)
									if _, ok := current[str]; !ok {
										m := map[string]any{}
										current[str] = m
									}
									ks = current[str]
								}
							}
							exprs = append(exprs, clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: keySetToStr(cols, keySet)}}})
							continue
						}
					}
				}
				exprs = append(exprs, cond)
			}
			if len(exprs) > 0 {
				res := tx.Clauses(clause.Where{Exprs: exprs}).Scan(dest)
				if res.Error != nil {
					db.AddError(res.Error)
					return
				}
				v := reflect.ValueOf(dest).Elem()
				l := v.Len()
				for i := 0; i < l; i++ {
					res := db.Session(&gorm.Session{NewDB: true}).Delete(v.Index(i).Addr().Interface())
					if res.Error != nil {
						db.AddError(res.Error)
						return
					}
				}
			} else {
				db.AddError(errors.New("could not delete " + rel.FieldSchema.Name + " related to " + rel.Schema.Name))
				return
			}
		}
	}
}

func keySetToStr(cols []clause.Column, vals any) string {
	result := ""
	if len(cols) == 1 {
		m := vals.(map[string]struct{})
		var keys string
		for v := range m {
			keys += v + ","
		}
		keys = strings.TrimSuffix(keys, ",")
		result += cols[0].Table + "." + cols[0].Name + " IN (" + keys + ")"
	} else {
		m := vals.(map[string]any)
		if len(m) > 1 {
			result += "("
		}
		first := true
		for k, v := range m {
			if first {
				first = false
			} else {
				result += ") OR ("
			}
			result += cols[0].Table + "." + cols[0].Name + " = " + k + " AND " + keySetToStr(cols[1:], v)
		}
		if len(m) > 1 {
			result += ")"
		}
	}
	return result
}
