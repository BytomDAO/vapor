package orm

import "time"

type AccessToken struct {
	ID      string `gorm:"primary_key"`
	Token   string
	Type    string
	Created time.Time
}
