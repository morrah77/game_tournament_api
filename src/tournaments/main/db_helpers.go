package main

import (
	"errors"
)

func fetchTournament(id uint) (tournament *Tournament, err error) {
	db.First(&tournament, id)
	if tournament.ID == 0 {
		return nil, errors.New("Tournament not found")
	}
	return tournament, nil
}

func fetchBalance(id uint) (balance *UserPointsBalance, err error) {
	db.Where(&UserPointsBalance{UserId: id}).First(&balance)
	if balance.ID == 0 {
		return nil, errors.New("Balance not found")
	}
	return balance, nil
}

func fetchBalances(ids ...[]uint) (balances []*UserPointsBalance, err error) {
	db.Where("UserId IN (?)", ids).Find(&balances)
	if len(balances) == 0 {
		return nil, errors.New("Balances not found")
	}
	return balances, nil
}

func joinTournamentAndTakePointsFromUserBalances(tournament *Tournament, joinTournamentRequest *JoinTournamentRequest) error {
	var (
		stakeholderIds []uint
		balances       []*UserPointsBalance
	)

	stakeholderIds = append(joinTournamentRequest.BackerIds, joinTournamentRequest.PlayerId)
	stakesCount := len(stakeholderIds)
	stake := tournament.Deposit / stakesCount

	tx := db.Begin()

	tx.Where("user_id IN (?)", stakeholderIds).Find(balances)
	if len(balances) == 0 {
		tx.Rollback()
		return errors.New("Balances not found")
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
