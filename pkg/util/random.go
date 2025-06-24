package util

import (
	"math/rand"
	"time"
)

var runesofrandom = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func Generaterandomstring(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = runesofrandom[r.Intn(len(runesofrandom))]
	}
	return string(b)
}
