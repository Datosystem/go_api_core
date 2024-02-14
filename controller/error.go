package controller

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
)

var EnableRecover bool

type Response struct {
	Data  interface{}
	Next  string
	Count int64
}

func AbortIfError(c *gin.Context, err error) bool {
	if err != nil {
		AbortWithError(c, err)
	}
	return err != nil
}

func AbortWithError(c *gin.Context, err error) {
	Hooks.AbortWithError.Run(c, err)
}

func RecoverIfEnabled(c *gin.Context) {
	if EnableRecover {
		Recover(c)
	}
}

func Recover(c *gin.Context) {
	if err := recover(); err != nil {
		stck := strings.Split(string(debug.Stack()), "\n")
		errStr := fmt.Sprint(err) + "\n" + strings.Join(stck[3:], "\n")
		Hooks.OnRecover.Run(c, errStr)
	}
}
