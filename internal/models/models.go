package models

import "time"

type User struct {
	ID           string
	Login        string
	PasswordHash string
	CreatedAt    time.Time
}

type Document struct {
	ID        string    `json:"id"`
	Owner     string    `json:"owner"`
	Name      string    `json:"name"`
	Mime      string    `json:"mime"`
	File      bool      `json:"file"`
	Public    bool      `json:"public"`
	CreatedAt time.Time `json:"created"`
	Grants    []string  `json:"grants"`
	JSONRaw   []byte    `json:"-"`
}

type DocumentMeta struct {
	Name   string   `json:"name"`
	Mime   string   `json:"mime"`
	File   bool     `json:"file"`
	Public bool     `json:"public"`
	Token  string   `json:"token"`
	Grants []string `json:"grants"`
}
