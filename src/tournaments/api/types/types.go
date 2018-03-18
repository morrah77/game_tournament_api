package types

import "time"

type ApiStorage interface {
	FetchTournament(uint) (interface{}, error)
	FetchTournaments(int, int) (interface{}, error)
	FetchBalance(uint) (interface{}, error)
	TakeAwayBalance(uint, int) (interface{}, error)
	TopUpBalance(uint, int) (interface{}, error)
	CreateNewTournament(*AnnounceTournamentRequest) (interface{}, error)
	JoinTournamentAndTakePointsFromUserBalances(*JoinTournamentRequest) error
	CheckAndSpreadTournamentPrize(*ResultTournamentRequest) error
}

type Tournament struct {
	ID        uint       `json:"id,omitempty"`
	CreatedAt time.Time  `json:"created_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	Date      time.Time  `json:"date,omitempty"`
	Deposit   int        `json:"deposit"` // let's don't use float32 to bonus points!
	GameId    int        `json:"game_id,omitempty"`
	State     uint
}

type UserPointsBalance struct {
	ID        uint       `json:"id,omitempty"`
	CreatedAt time.Time  `json:"created_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	UserId    uint
	Balance   int
}

type BalanceOperationRequest struct {
	PlayerId uint `json:"player_id"`
	Points   int  `json:"points"`
}

type AnnounceTournamentRequest struct {
	Date    time.Time `json:"date,omitempty"`
	Deposit int       `json:"deposit"` // let's don't use float32 to bonus points!
	GameId  int       `json:"game_id,omitempty"`
}

type JoinTournamentRequest struct {
	TournamentId uint   `json:"tournament_id"`
	PlayerId     uint   `json:"player_id"`
	BackerIds    []uint `json:"backer_ids,omitempty"`
}

type TournamentWinnerRequest struct {
	PlayerId uint `json:"player_id"`
	Prize    int  `json:"prize"`
}

type ResultTournamentRequest struct {
	TournamentId uint                       `json:"tournament_id"`
	Winners      []*TournamentWinnerRequest `json:"winners"`
}
