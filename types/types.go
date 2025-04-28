package types

import "time"

type ListedObject struct {
	Key          string    `json:"key"`
	LastModified time.Time `json:"last_modified"`
}
