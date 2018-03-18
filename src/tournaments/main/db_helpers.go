package main

import (
	"errors"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func fetchTournament(id uint) (tournament *Tournament, err error) {
	tournament = &Tournament{}
	if err = db.First(tournament, id).Error; err != nil {
		return nil, errors.New("An error occured during tournament fetching")
	}
	if tournament == nil {
		return nil, errors.New("Tournament not found")
	}
	return tournament, nil
}

func fetchTournaments(limit, offset int) (tournaments []*Tournament, err error) {
	tournaments = []*Tournament{}
	if err = db.Debug().Limit(limit).Offset(offset).Find(&tournaments).Error; err != nil {
		return nil, errors.New("An error occured during tournaments fetching")
	}
	if len(tournaments) == 0 {
		return nil, errors.New("Tournaments not found")
	}
	return tournaments, nil
}

func fetchBalance(id uint) (balance *UserPointsBalance, err error) {
	balance = &UserPointsBalance{}
	if err = db.Where(&UserPointsBalance{UserId: id}).First(&balance).Error; err != nil {
		return nil, errors.New("An error occured during Balance fetching")
	}
	if balance == nil {
		return nil, errors.New("Balance not found")
	}
	return balance, nil
}

func finishTransaction(tx *gorm.DB, err error) {
	if err != nil {
		logger.Printf("Rollback transaction due to error %#v\n", err.Error())
		if rbErr := tx.Rollback().Error; rbErr != nil {
			logger.Printf("An error occured duting transaction rollback: %s", rbErr.Error())
		}
		logger.Println("Transaction rolled back successfully")
	}
}

func topUpBalance(id uint, points int) (balance *UserPointsBalance, err error) {
	balance = &UserPointsBalance{}
	tx := db.Begin()
	defer func() { finishTransaction(tx, err) }()

	if err = tx.FirstOrInit(balance, &UserPointsBalance{UserId: id}).Error; err != nil {
		return nil, err
	}
	if balance == nil {
		balance = &UserPointsBalance{
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

func takeAwayBalance(id uint, points int) (balance *UserPointsBalance, err error) {
	balance = &UserPointsBalance{}
	tx := db.Begin()
	defer func() { finishTransaction(tx, err) }()

	if err = tx.FirstOrInit(balance, &UserPointsBalance{UserId: id}).Error; err != nil {
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

func createNewTournament(announceTournamentRequest *AnnounceTournamentRequest) (err error, tournament *Tournament) {
	// TODO(h.lazar) add game IDs somethere
	if announceTournamentRequest.GameId <= 0 {
		announceTournamentRequest.GameId = 1
	}
	tournament = &Tournament{
		Deposit: announceTournamentRequest.Deposit,
		Date:    announceTournamentRequest.Date,
		GameId:  announceTournamentRequest.GameId,
	}
	err = db.Save(tournament).Error
	return err, tournament
}

func joinTournamentAndTakePointsFromUserBalances(joinTournamentRequest *JoinTournamentRequest) (err error) {
	var (
		tournament     *Tournament
		stakeholderIds []uint
		balances       []*UserPointsBalance
	)

	stakeholderIds = make([]uint, 0)
	stakeholderIds = append(stakeholderIds, joinTournamentRequest.PlayerId)
	stakeholderIds = append(stakeholderIds, joinTournamentRequest.BackerIds...)
	stakesCount := len(stakeholderIds)
	if stakesCount <= 0 {
		err = errors.New(`Too few participants!`)
		return err
	}

	tx := db.Begin()
	tx.LogMode(true)
	defer func() { finishTransaction(tx, err) }()

	// TODO (h.lazar) add a check to all users be unique (do not allow user to back himself)
	tournament = &Tournament{}
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
	balances = []*UserPointsBalance{}

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

func checkAndSpreadTournamentPrize(resultTournamentRequest *ResultTournamentRequest) (err error) {
	var (
		tournament        *Tournament
		balances          []*UserPointsBalance
		tournamentPlayer  *TournamentPlayer
		tournamentBackers []*TournamentBacker
		stakeholderIds    []uint
	)

	tx := db.Begin()
	tx.LogMode(true)
	defer func() { finishTransaction(tx, err) }()

	tournament = &Tournament{}
	if err = tx.First(tournament, resultTournamentRequest.TournamentId).Error; err != nil {
		return err
	}
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

		balances = []*UserPointsBalance{}

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
				&UserPointsBalance{
					Balance: balance.Balance + stake,
				}).Error; err != nil {
				return err
			}
		}
	}

	if err = tx.Model(tournament).Update(&Tournament{State: 1}).Error; err != nil {
		return err
	}

	if err = tx.Commit().Error; err != nil {
		return err
	}
	return nil
}
