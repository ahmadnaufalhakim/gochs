package entity

import "time"

type LichessAccount struct {
	ID              string
	Username        string
	Title           string
	Patron          bool
	Online          bool
	Bullet          int
	Blitz           int
	Rapid           int
	Classical       int
	Puzzle          int
	Games           int
	Wins            int
	Draws           int
	Losses          int
	PlayTime        time.Duration
	MemberSince     time.Time
	FromEnvironment bool
}
