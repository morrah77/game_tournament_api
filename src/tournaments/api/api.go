package api

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/morrah77/game_tournament_api/src/tournaments/api/types"
)

type ApiConf struct {
	ListenAddr   string
	RelativePath string
}

type Api struct {
	conf   *ApiConf
	engine *gin.Engine
	stor   types.ApiStorage
	logger *log.Logger
}

func NewApi(conf *ApiConf, s interface{}, logger *log.Logger) (a *Api, err error) {
	//stor, ok := s.(types.ApiStorage)
	//if !ok {
	//	return nil, errors.New(`Unacceptable storage passed!`)
	//}
	a = &Api{
		conf: conf,
		// Let gin use its Logger() && Recovery() by default
		engine: gin.Default(),
		stor:   s.(types.ApiStorage),
		logger: logger,
	}
	a.mountRoutes(a.engine.Group(conf.RelativePath))
	return a, nil
}

func (a *Api) Run() error {
	return a.engine.Run(a.conf.ListenAddr)
}

func (a *Api) mountRoutes(api *gin.RouterGroup) {
	apiUser := api.Group("/user")
	apiUser.GET("/balance", a.getUserBalance)
	apiUser.POST("/take", a.takePointsFromUser)
	apiUser.POST("/fund", a.fundUserWithPoints)

	apiTournament := api.Group("/tournament")
	apiTournament.GET("/list", a.getTournaments)
	apiTournament.GET("/info", a.getTournamentInfo)
	apiTournament.POST("/announceTournament", a.announceTournament)
	apiTournament.POST("/joinTournament", a.joinTournament)
	apiTournament.POST("/resultTournament", a.resultTournament)
}
