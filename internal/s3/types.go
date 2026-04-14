package s3client

import "time"

type EntryKind int

const (
	KindPrefix EntryKind = iota
	KindObject
)

type Entry struct {
	Name         string
	FullKey      string
	Size         int64
	LastModified time.Time
	StorageClass string
	Kind         EntryKind
}
