package models

import "time"

type File struct {
	ID        int
	MessgaeID int
	FileID    string
	Type      string
	Size      int64
	CreatedAt time.Time
}
