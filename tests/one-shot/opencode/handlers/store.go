package handlers

import (
	"oauth2proxy/store"
)

func NewInMemoryStateStore() StateStore {
	return store.NewSessionStore()
}

func NewInMemoryUserStore() UserStore {
	return store.NewUserStore()
}

func NewInMemorySessionStore() SessionStore {
	return store.NewSessionStore()
}
