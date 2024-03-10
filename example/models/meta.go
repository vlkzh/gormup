package models

import (
	"database/sql/driver"
	"encoding/json"
)

type Meta map[string]any

func (m *Meta) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	result := Meta{}
	err := json.Unmarshal(bytes, &result)
	if err != nil {
		return err
	}
	*m = result

	return err
}

func (m Meta) Value() (driver.Value, error) {
	b, err := json.Marshal(m)
	return string(b), err
}

func (Meta) GormDataType() string {
	return "jsonb"
}
