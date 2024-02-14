package callbacks

import (
	"github.com/Datosystem/go_api_core/app"
	"gorm.io/gorm"
)

func RegisterGlobalModelHooks(db *gorm.DB, prefix string) {
	/*db.Callback().Create().After("gorm:after_create").Register(prefix+":global_after_create", CheckSkipDeleteCallback)*/
	db.Callback().Update().After("gorm:after_update").Register(prefix+":global_after_update", GlobalAfterUpdate)
}

func GlobalAfterUpdate(db *gorm.DB) {
	hook := app.GetModelHook(db.Statement.Table, app.AfterUpdateHook)
	if hook != nil {
		hook.Run(db)
	}
}
