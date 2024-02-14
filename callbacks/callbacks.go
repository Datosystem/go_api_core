package callbacks

import (
	"gorm.io/gorm"
)

// Register registers all available callbacks to *gorm.DB
func Register(db *gorm.DB, prefix string) {
	RegisterRecursiveDelete(db, prefix)
	RegisterCheckSkipDelete(db, prefix)
	RegisterGlobalModelHooks(db, prefix)
}
