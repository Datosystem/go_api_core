package message

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type Message interface {
	Text(string) Message
	ToMap() map[string]interface{}
	JSON(c *gin.Context)
	ToJSON() []byte
	Abort(c *gin.Context)
	Write(c *gin.Context)
	Error() string
	IsError() bool
	Is400() bool
	Is500() bool
	Set(key string, val interface{}) Message
	Get(key string) interface{}
	Add(msgs ...Message) Message
}

type Msg struct {
	Message    string
	Status     int
	Properties map[string]interface{}
}

func (m *Msg) Text(text string) Message {
	m.Message = text
	return m
}

func (m *Msg) ToMap() map[string]interface{} {
	mp := gin.H{"message": m.Message}
	if m.Properties != nil {
		for k, v := range m.Properties {
			mp[k] = v
		}
	}
	return mp
}

func (m *Msg) JSON(c *gin.Context) {
	c.JSON(m.Status, m.ToMap())
}

func (m *Msg) ToJSON() []byte {
	val, _ := json.Marshal(m.ToMap())
	return val
}

func (m *Msg) Abort(c *gin.Context) {
	c.AbortWithStatusJSON(m.Status, m.ToMap())
}

func (m *Msg) Error() string {
	return m.Message
}

func (m *Msg) Write(c *gin.Context) {
	if m.IsError() {
		m.Abort(c)
	} else {
		m.JSON(c)
	}
}

func (m *Msg) IsError() bool {
	return m.Is400() || m.Is500()
}

func (m *Msg) Is400() bool {
	return m.Status >= 400 && m.Status < 500
}

func (m *Msg) Is500() bool {
	return m.Status >= 500
}

func (m *Msg) Set(key string, val interface{}) Message {
	if m.Properties == nil {
		m.Properties = map[string]interface{}{}
	}
	m.Properties[key] = val
	return m
}

func (m *Msg) Get(key string) interface{} {
	if m.Properties == nil {
		return nil
	}
	return m.Properties[key]
}

func (m *Msg) Add(msgs ...Message) Message {
	for _, msg := range msgs {
		m.Message += "; " + msg.Error()
	}
	return m
}

func GetPrinter(c *gin.Context) (printer *message.Printer) {
	if c == nil {
		printer = message.NewPrinter(language.BritishEnglish)
	} else {
		printer = c.MustGet("i18n").(*message.Printer)
	}
	return printer
}

func FromError(status int, err error) Message {
	return &Msg{
		Message: err.Error(),
		Status:  status,
	}
}
