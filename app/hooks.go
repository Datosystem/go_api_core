package app

import "gorm.io/gorm"

type Hook[T any] struct {
	index *int
	Names []string
	Funcs []T
}

func (h *Hook[T]) Before(name string) *Hook[T] {
	for i, n := range h.Names {
		if n == name {
			h.index = &i
			return h
		}
	}
	h.index = nil
	return h
}

func (h *Hook[T]) After(name string) *Hook[T] {
	for i, n := range h.Names {
		if n == name {
			j := i + 1
			h.index = &j
			return h
		}
	}
	h.index = nil
	return h
}

func (h *Hook[T]) Index() int {
	if h.index == nil {
		return len(h.Names)
	} else {
		index := *h.index
		h.index = nil
		return index
	}
}

func (h *Hook[T]) Add(name string, hook T) *Hook[T] {
	index := h.Index()
	if len(h.Funcs) <= index {
		h.Names = append(h.Names, name)
		h.Funcs = append(h.Funcs, hook)
		return h
	}
	h.Names = append(h.Names[:index+1], h.Names[index:]...)
	h.Funcs = append(h.Funcs[:index+1], h.Funcs[index:]...)
	h.Names[index] = name
	h.Funcs[index] = hook
	return h
}

func (h *Hook[T]) Remove(name string) *Hook[T] {
	for i, n := range h.Names {
		if n == name {
			h.Names = append(h.Names[:i], h.Names[i+1:]...)
			h.Funcs = append(h.Funcs[:i], h.Funcs[i+1:]...)
			return h
		}
	}
	return h
}

type ModelHook struct {
	Hook[func(db *gorm.DB)]
}

func (h *ModelHook) Run(db *gorm.DB) {
	for _, fn := range h.Funcs {
		fn(db)
	}
}

type AppHooks struct {
	Models map[string]map[string]*ModelHook
}

func AddModelHook(modelName, hookType string, fn func(*gorm.DB)) {
	if _, ok := Hooks.Models[modelName]; !ok {
		Hooks.Models[modelName] = map[string]*ModelHook{}
	}
	if _, ok := Hooks.Models[modelName][hookType]; !ok {
		Hooks.Models[modelName][hookType] = &ModelHook{}
	}
	Hooks.Models[modelName][hookType].Add("", fn)
}

func GetModelHook(modelName, hookType string) *ModelHook {
	m, ok := Hooks.Models[modelName]
	if !ok {
		return nil
	}
	hook, ok := m[hookType]
	if !ok {
		return nil
	}
	return hook
}

type Webhook struct {
	ID_WEBHOOK int    `gorm:"primaryKey;type:int"`
	TYPE       string `gorm:"type:nvarchar(20)"`
	CONTEXT    string `gorm:"type:nvarchar(50)"`
	URL        string `gorm:"type:nvarchar(1000)"`
	METHOD     string `gorm:"type:varchar(5)"`
	QUERY_ARGS string `gorm:"type:ntext"`
	BODY       string `gorm:"type:ntext"`
}

func (Webhook) TableName() string {
	return "WEBHOOKS"
}
