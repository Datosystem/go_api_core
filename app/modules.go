package app

type KeySet struct {
	keys map[string]struct{}
}

// Has - Checks if all the @{keys} are present
func (s *KeySet) Has(key ...string) bool {
	for _, k := range key {
		if _, ok := s.keys[k]; !ok {
			return false
		}
	}
	return true
}

// HasOne - Checks if at least one of the @{keys} is present
func (s *KeySet) HasOne(keys ...string) bool {
	for _, k := range keys {
		if _, ok := s.keys[k]; ok {
			return true
		}
	}
	return false
}

func (s *KeySet) Add(key string) {
	s.keys[key] = struct{}{}
}

func (s *KeySet) Clear() {
	s.keys = map[string]struct{}{}
}
