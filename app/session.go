package app

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Datosystem/go_api_core/message"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SessionModel struct {
	KEY        string `gorm:"primaryKey"`
	EXPIRES_AT time.Time
	PROPERTIES string `gorm:"type:text"`
}

func (s SessionModel) TableName() string {
	return "SESSIONS"
}

type Session struct {
	properties map[string]interface{}
	expiresAt  time.Time
}

func (s *Session) Get(key string) interface{} {
	return s.properties[key]
}

func (s *Session) Set(key string, value interface{}) {
	s.properties[key] = value
}

func (s *Session) RefreshExpiration() {
	s.expiresAt = time.Now().Add(time.Hour * 12)
}

func (s *Session) SetExpired() {
	s.expiresAt = time.Now()
}

func (s *Session) IsExpired() bool {
	return s.expiresAt.Before(time.Now())
}

func (s *Session) Has(permissions ...string) bool {
	for _, perm := range permissions {
		if s.Get("PERMESSO_"+perm) != true {
			return false
		}
	}
	return true
}

func (s *Session) HasOne(permissions ...string) bool {
	for _, perm := range permissions {
		if s.Get("PERMESSO_"+perm) == true {
			return true
		}
	}
	return false
}

func (s *Session) Check(c *gin.Context, permissions ...string) message.Message {
	if !s.Has(permissions...) {
		return message.InsufficientPermissions(c, permissions...)
	}
	return nil
}

func (s *Session) CheckOne(c *gin.Context, permissions ...string) message.Message {
	if !s.HasOne(permissions...) {
		return message.InsufficientPermissionsHasOne(c, permissions...)
	}
	return nil
}

// Session providers

type sessionProvider interface {
	retrieve(key string) *Session
	store(key string, s *Session)
	delete(key string)
	clearExpired()
}

type inMemorySessionProvider struct {
	sessions map[string]*Session
}

func (sp *inMemorySessionProvider) retrieve(key string) *Session {
	return sp.sessions[key]
}

func (sp *inMemorySessionProvider) store(key string, s *Session) {
	if sp.sessions == nil {
		sp.sessions = make(map[string]*Session)
	}
	sp.sessions[key] = s
}

func (sp *inMemorySessionProvider) delete(key string) {
	delete(sp.sessions, key)
}

func (sp *inMemorySessionProvider) clearExpired() {
	for key, val := range sp.sessions {
		if val.IsExpired() {
			delete(sp.sessions, key)
		}
	}
}

type dbSessionProvider struct{}

func (sp *dbSessionProvider) retrieve(key string) *Session {
	session := SessionModel{}
	result := DB.Session(&gorm.Session{Logger: no404Logger}).First(&session, "\"KEY\" = ?", key)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}
	var properties map[string]interface{}
	json.Unmarshal([]byte(session.PROPERTIES), &properties)
	return &Session{properties, session.EXPIRES_AT}
}

func (sp *dbSessionProvider) store(key string, s *Session) {
	props, _ := json.Marshal(s.properties)
	session := SessionModel{
		KEY:        key,
		EXPIRES_AT: s.expiresAt,
		PROPERTIES: string(props),
	}
	DB.Session(&gorm.Session{Logger: no404Logger}).Save(session)
}

func (sp *dbSessionProvider) delete(key string) {
	DB.Where("\"KEY\" = ?", key).Delete(&SessionModel{})
}

func (sp *dbSessionProvider) clearExpired() {
	DB.Where("EXPIRES_AT < ?", time.Now()).Delete(&SessionModel{})
}

// Functions
func GetSession(c *gin.Context) *Session {
	return FindSession(strings.ReplaceAll(c.GetHeader("Authorization"), "Bearer ", ""))
}

func FindSession(key string) *Session {
	return provider.retrieve(key)
}

func CreateSession() *Session {
	clearExpired()
	s := &Session{properties: make(map[string]interface{})}
	s.RefreshExpiration()
	return s
}

func PutSession(key string, session *Session) {
	provider.store(key, session)
}

func DeleteSession(key string) {
	provider.delete(key)
}

func clearExpired() {
	provider.clearExpired()
}
