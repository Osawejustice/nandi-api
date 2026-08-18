package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONMap is a JSONB object stored on contacts, campaigns, and settings.
type JSONMap map[string]any

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value any) error {
	if value == nil {
		*m = JSONMap{}
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("jsonmap: unsupported type %T", value)
	}
	if len(raw) == 0 {
		*m = JSONMap{}
		return nil
	}
	return json.Unmarshal(raw, m)
}
