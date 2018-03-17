package main

import (
	"time"
)

const (
	USER_POINTS_OPERATION_DEBT = iota
	USER_POINTS_OPERATION_CREDIT
)

type Model struct {
	ID        uint       `gorm:"primary_key" json:"id,omitempty"`
	CreatedAt time.Time  `json:"created_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at,omitempty"`
	DeletedAt *time.Time `sql:"index" json:"deleted_at,omitempty"`
}

type User struct {
	Model
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Tournament struct {
	Model
	Date    time.Time `json:"date,omitempty"`
	Deposit int       `json:"deposit"` // let's don't use float32 to bonus points!
	GameId  int       `json:"game_id,omitempty"`
	State   uint
}

type TournamentPlayer struct {
	Model
	TournamentId uint
	UserId       uint
	UserDeposit  int
}

type TournamentBacker struct {
	Model
	TournamentId  uint
	UserId        uint
	BackerId      uint
	BackerDeposit int
}

type TournamentWinner struct {
	Model
	TournamentId uint
	UserId       uint
	Prize        int
}

type UserAuth struct {
	Model
	UserId uint
}

type UserPointsOperations struct {
	Model
	UserId        uint
	OperationType uint
	Sum           int
}

type UserPointsBalance struct {
	Model
	UserId  uint
	Balance int
}
