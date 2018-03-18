package api

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morrah77/go-developer-test-task-2/src/tournaments/api/types"
)

//Seek by HTTP query "id" param
//responds 400 on empty id, 404 on absent record,
//200 with full Tournament as "data" otherwise
func (a *Api) getTournamentInfo(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect ID provided"))
		return
	}
	intId, err := strconv.Atoi(id)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect ID provided"))
		a.logger.Println(err.Error())
		return
	}
	tournament, err := a.stor.FetchTournament(uint(intId))
	if err != nil {
		a.logger.Println(err.Error())
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Tournament not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": tournament.(*types.Tournament)})
}

//Fetch tournaments list
//accepts "limit" and "offset" HTTP query params
//responds 404 on absent records,
//200 with full Tournaments list as "data" otherwise
func (a *Api) getTournaments(ctx *gin.Context) {
	var (
		intLimit  int
		intOffset int
		err       error
	)
	limit := ctx.DefaultQuery("limit", "20")
	intLimit, err = strconv.Atoi(limit)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect limit provided"))
		a.logger.Println(err.Error())
		return
	}
	offset := ctx.DefaultQuery("offset", "0")
	intOffset, err = strconv.Atoi(offset)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect offset provided"))
		a.logger.Println(err.Error())
		return
	}

	tournaments, err := a.stor.FetchTournaments(intLimit, intOffset)
	if err != nil {
		a.logger.Println(err.Error())
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Tournaments not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": tournaments.([]*types.Tournament)})
}

//Seek by HTTP query "id" param
//responds 400 on empty id, 404 on absent record,
//200 with full UserPointsBalance as "data" otherwise
func (a *Api) getUserBalance(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect ID provided"))
		return
	}
	intId, err := strconv.Atoi(id)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect ID provided"))
		a.logger.Println(err.Error())
		return
	}
	balance, err := a.stor.FetchBalance(uint(intId))
	if err != nil {
		a.logger.Println(err.Error())
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Balance not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": balance.(*types.UserPointsBalance)})
}

//processes POST JSON body like {"player_id":1,"points":100}
//requires "player_id", "points" fields,
//responds 500 on error, 200 with full UserPointsBalance otherwise
func (a *Api) takePointsFromUser(ctx *gin.Context) {
	var parsedRequestBody types.BalanceOperationRequest
	data, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errors.New("Could not read request body"))
		a.logger.Println(err.Error())
		return
	}
	if err := json.Unmarshal(data, &parsedRequestBody); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect request body provided"))
		a.logger.Println(err.Error())
		return
	}
	playerId := parsedRequestBody.PlayerId
	if playerId == 0 {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect player ID provided"))
		return
	}
	points := parsedRequestBody.Points
	if points <= 0 {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect points value provided"))
		return
	}
	balance, err := a.stor.TakeAwayBalance(playerId, points)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Balance not taken away"})
		a.logger.Println(err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": balance.(*types.UserPointsBalance)})
}

//processes POST JSON body like {"player_id":1,"points":100}
//requires "player_id", "points" fields,
//responds 500 on error, 200 with full UserPointsBalance otherwise
func (a *Api) fundUserWithPoints(ctx *gin.Context) {
	var parsedRequestBody types.BalanceOperationRequest
	data, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errors.New("Could not read request body: "+err.Error()))
		a.logger.Println(err.Error())
		return
	}
	if err := json.Unmarshal(data, &parsedRequestBody); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect request body provided: "+err.Error()))
		a.logger.Println(err.Error())
		return
	}
	playerId := parsedRequestBody.PlayerId
	if playerId == 0 {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect player ID provided"))
		return
	}
	points := parsedRequestBody.Points
	if points <= 0 {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect points value provided"))
		return
	}
	balance, err := a.stor.TopUpBalance(playerId, points)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Balance not replenished"})
		a.logger.Println(err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": balance.(*types.UserPointsBalance)})
}

//processes POST JSON body like {"deposit":100}, {"deposit":100,"game_id":1}, {"date":"2018-03-18T00:59:00Z","deposit":100,"game_id":1}
//requires "deposit" field,
//accepts "date" and "gameId", fills by default current date and 0 appropriately,
//responds 500 on error, 200 with full Tournament otherwise
func (a *Api) announceTournament(ctx *gin.Context) {
	var (
		parsedRequestBody types.AnnounceTournamentRequest
		tournament        interface{}
	)
	data, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errors.New("Could not read request body"))
		a.logger.Println(err.Error())
		return
	}
	if err := json.Unmarshal(data, &parsedRequestBody); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect request body provided"))
		a.logger.Println(err.Error())
		return
	}
	if parsedRequestBody.Deposit <= 0 {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect tournament deposit provided"))
		return
	}
	// TODO(h.lazar) improve this check, pay attention to timezone
	if parsedRequestBody.Date.IsZero() {
		parsedRequestBody.Date = time.Now()
	} else if parsedRequestBody.Date.Before(time.Now()) {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect tournament date provided"))
		return
	}

	tournament, err = a.stor.CreateNewTournament(&parsedRequestBody)
	if err != nil {
		a.logger.Println(err.Error())
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "could not announce tournament"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": tournament.(*types.Tournament)})
}

//processes POST JSON body like {"tournament_id"1,"player_id":2}, {"tournament_id"1,"player_id":2,"backer_ids":[3,4,5]}
//requires "tournament_id", "player_id" fields,
//accepts "backer_ids",
//responds 500 on error, 204 otherwise
func (a *Api) joinTournament(ctx *gin.Context) {
	var parsedRequestBody types.JoinTournamentRequest
	data, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errors.New("Could not read request body"))
		a.logger.Println(err.Error())
		return
	}
	if err := json.Unmarshal(data, &parsedRequestBody); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect request body provided"))
		a.logger.Println(err.Error())
		return
	}
	err = a.stor.JoinTournamentAndTakePointsFromUserBalances(&parsedRequestBody)
	if err != nil {
		a.logger.Println(err.Error())
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "could not join tournament"})
		return
	}

	ctx.String(http.StatusNoContent, ``)
}

//processes POST JSON body like {"tournament_id":1,"winners":[{"player_id":1,"prize":500}]}
//requires "tournament_id", "winners" fields,
//responds 500 on error, 204 otherwise
func (a *Api) resultTournament(ctx *gin.Context) {
	var parsedRequestBody types.ResultTournamentRequest
	data, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errors.New("Could not read request body"))
		a.logger.Println(err.Error())
		return
	}
	if err := json.Unmarshal(data, &parsedRequestBody); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Incorrect request body provided"))
		a.logger.Println(err.Error())
		return
	}
	err = a.stor.CheckAndSpreadTournamentPrize(&parsedRequestBody)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Could not save tournament result"})
		a.logger.Println(err.Error())
		return
	}

	ctx.String(http.StatusNoContent, ``)
}
