package main

import (
	"fmt"
	"hash/maphash"
	"strings"

	"github.com/satorunooshie/equiv"
	"github.com/satorunooshie/equiv/hashers"
)

type User struct {
	Tenant string
	Email  string
	Name   string
}

func main() {
	// Only tenant and normalized email participate in identity; Name is a value.
	h := hashers.Struct[User]().
		Field(func(u User) string { return u.Tenant }, maphash.ComparableHasher[string]{}).
		Field(func(u User) string { return strings.ToLower(u.Email) }, maphash.ComparableHasher[string]{}).
		Build()
	m := equiv.NewMap[User, string](h)
	m.Set(User{Tenant: "acme", Email: "Alice@EXAMPLE.COM", Name: "Alice"}, "enabled")
	v, ok := m.Get(User{Tenant: "acme", Email: "alice@example.com", Name: "other"})
	fmt.Println(v, ok)
}
