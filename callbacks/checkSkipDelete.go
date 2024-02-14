package callbacks

import "gorm.io/gorm"

func RegisterCheckSkipDelete(db *gorm.DB, prefix string) {
	db.Callback().Delete().Register(prefix+":check_skip_delete", CheckSkipDeleteCallback)
}

func CheckSkipDeleteCallback(db *gorm.DB) {
	if db.Error != nil && db.Error.Error() == "___SKIP_DELETE___" {
		db.Error = nil
	}
}
