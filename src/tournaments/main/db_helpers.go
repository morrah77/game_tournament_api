package main

import (
	"errors"
)

func fetchTournament(id uint) (tournament *Tournament, err error) {
	tournament = &Tournament{}
	if err = db.First(tournament, id).Error; err != nil {
		logger.Println(err.Error())
		return nil, errors.New("An error occured during tournament fetching")
	}
	if tournament == nil {
		return nil, errors.New("Tournament not found")
	}
	return tournament, nil
}

func fetchBalance(id uint) (balance *UserPointsBalance, err error) {
	balance = &UserPointsBalance{}
	if err = db.Where(&UserPointsBalance{UserId: id}).First(&balance).Error; err != nil {
		logger.Println(err.Error())
		return nil, errors.New("An error occured during Balance fetching")
	}
	if balance == nil {
		return nil, errors.New("Balance not found")
	}
	return balance, nil
}

func fetchBalances(ids ...[]uint) (balances []*UserPointsBalance, err error) {
	if err = db.Where("UserId IN (?)", ids).Find(&balances).Error; err != nil {
		logger.Println(err.Error())
		return nil, errors.New("An error occured during Balances fetching")
	}
	if len(balances) == 0 {
		return nil, errors.New("Balances not found")
	}
	return balances, nil
}

func topUpBalance(id uint, points int) (balance *UserPointsBalance, err error) {
	balance = &UserPointsBalance{}
	tx := db.Begin()

	if err = tx.Where(&UserPointsBalance{UserId: id}).First(balance).Error; err != nil {
		tx.Rollback()
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
		OperationType: 1,
		Sum:           points,
	}
	if err = tx.Save(balance).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err = tx.Create(operation).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err = tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	return balance, nil
}

func takeAwayBalance(id uint, points int) (balance *UserPointsBalance, err error) {
	balance = &UserPointsBalance{}
	tx := db.Begin()

	if err = tx.Where(&UserPointsBalance{UserId: id}).First(balance).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if balance == nil || balance.Balance < points {
		tx.Rollback()
		return nil, errors.New(`Not enough points in user balance!`)
	}
	balance.Balance -= points
	operation := &UserPointsOperations{
		UserId:        id,
		OperationType: 2,
		Sum:           points,
	}
	if err = tx.Save(balance).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err = tx.Create(operation).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err = tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	return balance, nil
}

func joinTournamentAndTakePointsFromUserBalances(tournament *Tournament, joinTournamentRequest *JoinTournamentRequest) (err error) {
	var (
		stakeholderIds []uint
		balances       []*UserPointsBalance
	)

	stakeholderIds = append(joinTournamentRequest.BackerIds, joinTournamentRequest.PlayerId)
	stakesCount := len(stakeholderIds)
	stake := tournament.Deposit / stakesCount

	tx := db.Begin()

	if err = tx.Where("user_id IN (?)", stakeholderIds).Find(balances).Error; err != nil {
		logger.Println(err.Error())
		return errors.New("An error occured during users' balances fetching")
	}
	if len(balances) == 0 {
		tx.Rollback()
		return errors.New("Users' balances not found")
	}

	if len(balances) < len(stakeholderIds) {
		return errors.New("One or more participants have no balance or user backs himself")
	}

	for _, balance := range balances {
		if balance.Balance < stake {
			tx.Rollback()
			return errors.New("One or more participants have not enough balance")
		}
		if balance.UserId == joinTournamentRequest.PlayerId {
			tx.Create(
				&TournamentPlayer{
					TournamentId: joinTournamentRequest.TournamentId,
					UserId:       joinTournamentRequest.PlayerId,
					UserDeposit:  stake,
				})
		} else {
			tx.Create(
				&TournamentBacker{
					TournamentId:  joinTournamentRequest.TournamentId,
					UserId:        joinTournamentRequest.PlayerId,
					BackerId:      balance.UserId,
					BackerDeposit: stake,
				})
		}
		tx.Create(&UserPointsOperations{UserId: balance.UserId, OperationType: 1, Sum: stake})
		tx.Where(balance.ID).Update(&UserPointsBalance{UserId: balance.UserId, Balance: balance.Balance - stake})
	}

	tx.Commit()
	return nil
}

func checkAndSpreadTournamentPrize(tournament *Tournament, winners []*TournamentWinnerRequest) error {
	var (
		balances          []*UserPointsBalance
		tournamentPlayer  *TournamentPlayer
		tournamentBackers []*TournamentBacker
		stakeholderIds    []uint
	)

	tx := db.Begin()

	for _, winner := range winners {
		db.Where(&TournamentPlayer{UserId: winner.PlayerId}).First(tournamentPlayer)
		if tournamentPlayer.ID == 0 {
			return errors.New("Tournament player not found")
		}

		tx.Create(
			&TournamentWinner{
				TournamentId: tournament.ID,
				UserId:       winner.PlayerId,
				Prize:        winner.Prize,
			})

		stakeholderIds = []uint{winner.PlayerId}

		tx.Where("user_id=?", winner.PlayerId).Find(tournamentBackers)
		for _, backer := range tournamentBackers {
			stakeholderIds = append(stakeholderIds, backer.BackerId)
		}

		tx.Where("user_id IN (?)", stakeholderIds).Find(balances)
		if len(balances) < len(stakeholderIds) {
			return errors.New("One or more participants have no balance")
		}

		stakesCount := len(stakeholderIds)
		stake := tournament.Deposit / stakesCount

		for _, balance := range balances {
			tx.Create(&UserPointsOperations{UserId: balance.UserId, OperationType: 0, Sum: stake})
			tx.Where(balance.ID).Update(&UserPointsBalance{UserId: balance.UserId, Balance: balance.Balance + stake})
		}
	}

	tx.Commit()
	return nil
}
