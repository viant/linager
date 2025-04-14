package graph

import (
	"github.com/minio/highwayhash"
)

var key = []byte("0123456789ABCDEF0123456789ABCDEF")

func Hash(data []byte) (uint64, error) {
	hash, err := highwayhash.New64(key)
	if err != nil {
		return 0, err
	}
	_, err = hash.Write(data)
	return hash.Sum64(), err
}
