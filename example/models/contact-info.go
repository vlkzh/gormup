package models

import (
	"database/sql/driver"
	"encoding/json"
)

type ContactInfo struct {
	Address string
	Point   []int
	Phone   string
}

func (m *ContactInfo) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	result := ContactInfo{}
	err := json.Unmarshal(bytes, &result)
	if err != nil {
		return err
	}
	*m = result

	return err
}

func (m ContactInfo) Value() (driver.Value, error) {
	b, err := json.Marshal(m)
	return string(b), err
}

func (ContactInfo) GormDataType() string {
	return "jsonb"
}
