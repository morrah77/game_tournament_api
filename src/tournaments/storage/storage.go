package storage

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"

	"github.com/morrah77/game_tournament_api/src/tournaments/api/types"
)

const MAX_CONNECTION_ATTEMPTS = 10
const CONNECTION_ATTEMPTS_INTERVAL_SECONDS = 5

type DsnColfig struct {
	DbHost string
	DbPort string
	DbUser string
	DbPass string
	DbName string
}

type Storage struct {
	db     *gorm.DB
	logger *log.Logger
}

func NewStorage(conf *DsnColfig, logger *log.Logger) (interface{}, error) {
	var (
		db  *gorm.DB
		err error
		dsn string
	)
	dsn = getConnectionString(conf)
	connectionAttempts := 0
	for {
		db, err = gorm.Open("postgres", dsn)
		if err == nil {
			logger.Print(`DB Connection success!`)
			break
		}
		logger.Print(err.Error())
		connectionAttempts++
		if connectionAttempts > MAX_CONNECTION_ATTEMPTS {
			return nil, errors.New("Could not connect to database!")
		}
		time.Sleep(CONNECTION_ATTEMPTS_INTERVAL_SECONDS * time.Second)
	}
	s := &Storage{
		db:     db,
		logger: logger,
	}
	s.autoMigrate()
	return s, nil
}

func (s *Storage) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Storage) autoMigrate() {
	s.db.AutoMigrate(
		&User{},
		&UserAuth{},
		&types.Tournament{},
		&TournamentPlayer{},
		&TournamentBacker{},
		&TournamentWinner{},
		&UserPointsOperations{},
		&types.UserPointsBalance{},
	)
}

func (s *Storage) FetchTournament(id uint) (interface{}, error) {
	var (
		tournament *types.Tournament
		err        error
	)
	tournament = &types.Tournament{}
	if err = s.db.First(tournament, id).Error; err != nil {
		return nil, errors.New("An error occured during tournament fetching")
	}
	if tournament == nil {
		return nil, errors.New("Tournament not found")
	}
	return tournament, nil
}

func (s *Storage) FetchTournaments(limit, offset int) (interface{}, error) {
	var (
		tournaments []*types.Tournament
		err         error
	)
	tournaments = []*types.Tournament{}
	if err = s.db.Debug().Limit(limit).Offset(offset).Find(&tournaments).Error; err != nil {
		return nil, errors.New("An error occured during tournaments fetching")
	}
	if len(tournaments) == 0 {
		return nil, errors.New("Tournaments not found")
	}
	return tournaments, nil
}

func (s *Storage) FetchBalance(id uint) (interface{}, error) {
	var (
		balance *types.UserPointsBalance
		err     error
	)
	balance = &types.UserPointsBalance{}
	if err = s.db.Where(&types.UserPointsBalance{UserId: id}).First(&balance).Error; err != nil {
		return nil, errors.New("An error occured during Balance fetching")
	}
	if balance == nil {
		return nil, errors.New("Balance not found")
	}
	return balance, nil
}

func (s *Storage) finishTransaction(tx *gorm.DB, err error) {
	if err != nil {
		s.logger.Printf("Rollback transaction due to error %#v\n", err.Error())
		if rbErr := tx.Rollback().Error; rbErr != nil {
			s.logger.Printf("An error occured duting transaction rollback: %s", rbErr.Error())
		}
		s.logger.Println("Transaction rolled back successfully")
	}
}

func (s *Storage) TopUpBalance(id uint, points int) (interface{}, error) {
	var (
		balance *types.UserPointsBalance
		err     error
	)
	balance = &types.UserPointsBalance{}
	tx := s.db.Begin()
	defer func() { s.finishTransaction(tx, err) }()

	if err = tx.FirstOrInit(balance, &types.UserPointsBalance{UserId: id}).Error; err != nil {
		return nil, err
	}
	if balance == nil {
		balance = &types.UserPointsBalance{
			UserId:  id,
			Balance: 0,
		}
	}
	balance.Balance += points
	operation := &UserPointsOperations{
		UserId:        id,
		OperationType: USER_POINTS_OPERATION_DEBT,
		Sum:           points,
	}
	if err = tx.Save(balance).Error; err != nil {
		return nil, err
	}
	if err = tx.Create(operation).Error; err != nil {
		return nil, err
	}
	if err = tx.Commit().Error; err != nil {
		return nil, err
	}
	return balance, nil
}

func (s *Storage) TakeAwayBalance(id uint, points int) (interface{}, error) {
	var (
		balance *types.UserPointsBalance
		err     error
	)
	balance = &types.UserPointsBalance{}
	tx := s.db.Begin()
	defer func() { s.finishTransaction(tx, err) }()

	if err = tx.FirstOrInit(balance, &types.UserPointsBalance{UserId: id}).Error; err != nil {
		return nil, err
	}
	if balance == nil || balance.Balance < points {
		return nil, errors.New(`Not enough points in user balance!`)
	}
	balance.Balance -= points
	operation := &UserPointsOperations{
		UserId:        id,
		OperationType: USER_POINTS_OPERATION_CREDIT,
		Sum:           points,
	}
	if err = tx.Save(balance).Error; err != nil {
		return nil, err
	}
	if err = tx.Create(operation).Error; err != nil {
		return nil, err
	}
	if err = tx.Commit().Error; err != nil {
		return nil, err
	}
	return balance, nil
}

func (s *Storage) CreateNewTournament(announceTournamentRequest *types.AnnounceTournamentRequest) (interface{}, error) {
	// TODO(h.lazar) add game IDs somethere
	if announceTournamentRequest.GameId <= 0 {
		announceTournamentRequest.GameId = 1
	}
	tournament := &types.Tournament{
		Deposit: announceTournamentRequest.Deposit,
		Date:    announceTournamentRequest.Date,
		GameId:  announceTournamentRequest.GameId,
	}
	err := s.db.Save(tournament).Error
	return tournament, err
}

func (s *Storage) JoinTournamentAndTakePointsFromUserBalances(joinTournamentRequest *types.JoinTournamentRequest) (err error) {
	var (
		tournament     *types.Tournament
		stakeholderIds []uint
		balances       []*types.UserPointsBalance
	)

	stakeholderIds = make([]uint, 0)
	stakeholderIds = append(stakeholderIds, joinTournamentRequest.PlayerId)
	stakeholderIds = append(stakeholderIds, joinTournamentRequest.BackerIds...)
	stakesCount := len(stakeholderIds)
	if stakesCount <= 0 {
		err = errors.New(`Too few participants!`)
		return err
	}

	tx := s.db.Begin()
	//tx.LogMode(true)
	defer func() { s.finishTransaction(tx, err) }()

	// TODO (h.lazar) add a check to all users be unique (do not allow user to back himself)
	tournament = &types.Tournament{}
	if err = tx.First(tournament, joinTournamentRequest.TournamentId).Error; err != nil {
		return err
	}
	if tournament.Date.Before(time.Now()) {
		err = errors.New(`Tournament out of date!`)
		return err
	}
	if tournament.State != 0 {
		err = errors.New(`Tournament already finished!`)
		return err
	}

	if !tx.Where(&TournamentPlayer{UserId: joinTournamentRequest.PlayerId, TournamentId: tournament.ID}).First(&TournamentPlayer{}).RecordNotFound() {
		err = errors.New(`User already perticipates tournament!`)
		return err
	}

	stake := tournament.Deposit / stakesCount
	balances = []*types.UserPointsBalance{}

	if err = tx.Where("user_id IN (?)", stakeholderIds).Find(&balances).Error; err != nil {
		return err
	}
	if len(balances) == 0 {
		err = errors.New("Users' balances not found")
		return err
	}

	if len(balances) < len(stakeholderIds) {
		err = errors.New("One or more participants have no balance or user backs himself")
		return err
	}

	for _, balance := range balances {
		if balance.Balance < stake {
			err = errors.New("One or more participants have not enough balance")
			return err
		}
		if balance.UserId == joinTournamentRequest.PlayerId {
			err = tx.Create(
				&TournamentPlayer{
					TournamentId: joinTournamentRequest.TournamentId,
					UserId:       joinTournamentRequest.PlayerId,
					UserDeposit:  stake,
				}).Error
		} else {
			err = tx.Create(
				&TournamentBacker{
					TournamentId:  joinTournamentRequest.TournamentId,
					UserId:        joinTournamentRequest.PlayerId,
					BackerId:      balance.UserId,
					BackerDeposit: stake,
				}).Error
		}
		if err != nil {
			return err
		}
		if err = tx.Create(
			&UserPointsOperations{
				UserId:        balance.UserId,
				OperationType: USER_POINTS_OPERATION_CREDIT,
				Sum:           stake,
			}).Error; err != nil {
			return err
		}
		if err = tx.Model(balance).Update(`balance`, balance.Balance-stake).Error; err != nil {
			return err
		}
	}

	if err = tx.Commit().Error; err != nil {
		return err
	}
	return nil
}

func (s *Storage) CheckAndSpreadTournamentPrize(resultTournamentRequest *types.ResultTournamentRequest) (err error) {
	var (
		tournament        *types.Tournament
		balances          []*types.UserPointsBalance
		tournamentPlayer  *TournamentPlayer
		tournamentBackers []*TournamentBacker
		stakeholderIds    []uint
	)

	tx := s.db.Begin()
	//tx.LogMode(true)
	defer func() { s.finishTransaction(tx, err) }()

	tournament = &types.Tournament{}
	if err = tx.First(tournament, resultTournamentRequest.TournamentId).Error; err != nil {
		return err
	}
	//TODO(h.lazar) commented just for testing conveniency. To be uncommented
	//if tournament.Date.After(time.Now()) {
	//	err = errors.New(`Tournament still did not started!`)
	//	return err
	//}
	if tournament.State != 0 {
		err = errors.New(`Tournament already finished!`)
		return err
	}

	for _, winner := range resultTournamentRequest.Winners {

		tournamentPlayer = &TournamentPlayer{}

		if err = tx.Where(
			&TournamentPlayer{
				UserId:       winner.PlayerId,
				TournamentId: tournament.ID,
			}).First(&tournamentPlayer).Error; err != nil {
			return err
		}

		if err = tx.Create(
			&TournamentWinner{
				TournamentId: tournament.ID,
				UserId:       winner.PlayerId,
				Prize:        winner.Prize,
			}).Error; err != nil {
			return err
		}

		stakeholderIds = []uint{winner.PlayerId}

		tournamentBackers = []*TournamentBacker{}

		if err = tx.Where(
			&TournamentBacker{
				UserId:       winner.PlayerId,
				TournamentId: tournament.ID,
			}).Find(&tournamentBackers).Error; err != nil {
			return err
		}
		for _, backer := range tournamentBackers {
			stakeholderIds = append(stakeholderIds, backer.BackerId)
		}

		balances = []*types.UserPointsBalance{}

		if err = tx.Where("user_id IN (?)", stakeholderIds).Find(&balances).Error; err != nil {
			return err
		}
		if len(balances) < len(stakeholderIds) {
			err = errors.New("One or more participants have no balance")
			return err
		}

		stakesCount := len(stakeholderIds)
		stake := winner.Prize / stakesCount

		for _, balance := range balances {
			if err = tx.Create(
				&UserPointsOperations{
					UserId:        balance.UserId,
					OperationType: USER_POINTS_OPERATION_DEBT,
					Sum:           stake,
				}).Error; err != nil {
				return err
			}
			if err = tx.Model(balance).Update(
				&types.UserPointsBalance{
					Balance: balance.Balance + stake,
				}).Error; err != nil {
				return err
			}
		}
	}

	if err = tx.Model(tournament).Update(&types.Tournament{State: 1}).Error; err != nil {
		return err
	}

	if err = tx.Commit().Error; err != nil {
		return err
	}
	return nil
}

func getConnectionString(conf *DsnColfig) string {
	// https://www.postgresql.org/docs/9.5/static/app-postgres.html
	return fmt.Sprintf("host=%#s port=%#s user=%#s password=%#s dbname=%#s sslmode=disable",
		conf.DbHost,
		conf.DbPort,
		conf.DbUser,
		conf.DbPass,
		conf.DbName,
	)
}
