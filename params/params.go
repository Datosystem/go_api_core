package params

import (
	"encoding/json"

	"github.com/Datosystem/go_api_core/message"
	"github.com/gin-gonic/gin"
	"github.com/iancoleman/orderedmap"
	"gorm.io/gorm/schema"
)

type NextNesterer[NestedType any] interface {
	NextNested() map[string]NestedType
}

type Conditions struct {
	// Types: N nested, I inner join, O outer join, 0 join, M mixed - nested and join
	Type   string
	Query  string
	Args   []interface{}
	Nested map[string]*Conditions
}

func (c *Conditions) NextNested() map[string]*Conditions {
	return c.Nested
}

func ToStmt(c *gin.Context, params, p string, modelSchema *schema.Schema, alias string, conds *Conditions, allowed map[string]struct{}) message.Message {
	if len(params) > 0 {
		var paramsArr []interface{}
		if json.Unmarshal([]byte(params), &paramsArr) != nil {
			return message.InvalidParamsJSON(c)
		}
		if msg := parseParams(c, modelSchema, alias, paramsArr, conds, allowed); msg != nil {
			return msg
		}
	}
	pMap := orderedmap.New()
	if len(p) > 0 {
		if json.Unmarshal([]byte(p), &pMap) != nil {
			return message.InvalidParamsJSON(c)
		}
	}
	// TODO: Remove in the future if no longer necessary
	/*if extraParams, exists := c.Get("P"); exists {
		if orderedMap, ok := extraParams.(*orderedmap.OrderedMap); ok {
			for _, key := range orderedMap.Keys() {
				val, _ := orderedMap.Get(key)
				p.Set(key, val)
			}
		}
	}*/

	if len(pMap.Keys()) > 0 {
		if msg := parseParamsV2(c, modelSchema, alias, pMap, conds, allowed); msg != nil {
			return msg
		}
	}
	return nil
}
