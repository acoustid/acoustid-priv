package priv

import "time"

type Cache interface {
	Set(key string, value interface{}, expire time.Duration)
	Get(key string) (interface{}, bool)
	Delete(key string)
}
