package app

import (
	"embed"
	"log"
	"os"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var Flags = KeySet{keys: map[string]struct{}{}}
var Properties = map[string]string{}
var Hooks = AppHooks{Models: map[string]map[string]*ModelHook{}}
var DB *gorm.DB
var FS *embed.FS
var provider = dbSessionProvider{}
var no404Logger = logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{SlowThreshold: 200 * time.Millisecond, Colorful: true, IgnoreRecordNotFoundError: true, LogLevel: logger.Warn})

const AfterUpdateHook = "AfterUpdate"
