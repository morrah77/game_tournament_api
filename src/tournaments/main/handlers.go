package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/bckp/log"
	"github.com/gin-gonic/gin"
)

/**
Seek by HTTP query "id" param
responds 400 on empty id, 404 on absent record,
200 with full Tournament as "data" otherwise
*/
func getTournamentInfo(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect ID provided"))
		return
	}
	intId, err := strconv.Atoi(id)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect ID provided"))
		return
	}
	tournament, err := fetchTournament(uint(intId))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Tournament not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": tournament})
}

func getUserBalance(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect ID provided"))
		return
	}
	intId, err := strconv.Atoi(id)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect ID provided"))
		return
	}
	balance, err := fetchBalance(uint(intId))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Balance not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": balance})
}

func takePointsFromUser(ctx *gin.Context) {
	//TODO(h.lazar) implement it!
	ctx.Render(http.StatusOK, nil)
}

func fundUserWithPoints(ctx *gin.Context) {
	//TODO(h.lazar) implement it!
	ctx.Render(http.StatusOK, nil)
}

/**
processes POST JSON body like {"deposit":100}, {"deposit":100,"game_id":1}, {"date":1515947534,"deposit":100,"game_id":1}
requires "deposit" field,
accepts "date" and "gameId", fills by default current date and 0 appropriately,
responds 500 on error, 200 with full Tournament otherwise
*/
func announceToutnament(ctx *gin.Context) {
	var tournament Tournament
	data, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errors.New("Could not read request body"))
		return
	}
	if err := json.Unmarshal(data, &tournament); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect request body provided"))
		return
	}
	if tournament.Deposit <= 0 {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect tournament deposit provided"))
		return
	}
	// TODO(h.lazar) improve this check, pay attention to timezone
	if tournament.Date.IsZero() {
		tournament.Date = time.Now()
	} else if tournament.Date.Before(time.Now()) {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect tournament time provided"))
		return
	}
	// TODO(h.lazar) add game IDs somethere
	if tournament.GameId <= 0 {
		tournament.GameId = 1
	}
	db.Save(tournament)
	if tournament.ID == 0 {
		ctx.AbortWithError(http.StatusInternalServerError, errors.New("Could not announce tournament"))
	}
	ctx.JSON(http.StatusOK, gin.H{"data": tournament})
}

type JoinTournamentRequest struct {
	TournamentId uint   `json:"tournament_id"`
	PlayerId     uint   `json:"player_id"`
	BackerIds    []uint `json:"backer_ids",omitempty`
}

/**
processes POST JSON body like {"tournament_id"1,"player_id":2}, {"tournament_id"1,"player_id":2,"backer_ids":[3,4,5]}
requires "tournament_id", "player_id" fields,
accepts "backer_ids",
responds 500 on error, 200 otherwise
*/
func joinTournament(ctx *gin.Context) {
	var parsedRequestBody JoinTournamentRequest
	data, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errors.New("Could not read request body"))
		return
	}
	if err := json.Unmarshal(data, &parsedRequestBody); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect request body provided"))
		return
	}
	// TODO (h.lazar) add a check to all users be unique (do not allow user to back himself)
	tournament, err := fetchTournament(parsedRequestBody.TournamentId)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Tournament not found"})
		return
	}
	err = joinTournamentAndTakePointsFromUserBalances(tournament, &parsedRequestBody)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "One or more participants consistency is not enough: "})
		log.Error(err)
		return
	}

	ctx.Render(http.StatusOK, nil)
}

type TournamentWinnerRequest struct {
	PlayerId uint `json:"player_id""`
	Prize    int  `json:"prize"`
}

type ResultTournamentRequest struct {
	TournamentId uint                       `json:"tournament_id"`
	Winners      []*TournamentWinnerRequest `json:"winners"`
}

func resultTournament(ctx *gin.Context) {
	var parsedRequestBody ResultTournamentRequest
	data, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errors.New("Could not read request body"))
		return
	}
	if err := json.Unmarshal(data, &parsedRequestBody); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect request body provided"))
		return
	}
	tournament, err := fetchTournament(parsedRequestBody.TournamentId)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Tournament not found"})
		return
	}

	err = checkAndSpreadTournamentPrize(tournament, parsedRequestBody.Winners)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "One or more participants consistency is not enough: "})
		log.Error(err)
		return
	}

	ctx.Render(http.StatusOK, nil)
}
