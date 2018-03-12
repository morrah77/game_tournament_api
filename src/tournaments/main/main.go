package main

import (
	"flag"
	"fmt"

	"github.com/bckp/log"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

var db *gorm.DB

var conf struct {
	dbhost string
	dbport string
	dbuser string
	dbpass string
	dbname string
	lsaddr string
}

func init() {
	flag.StringVar(&conf.dbhost, "db-host", "localhost", "Database host")
	flag.StringVar(&conf.dbport, "db-port", "5432", "Database port")
	flag.StringVar(&conf.dbuser, "db-user", "postgres", "Database username")
	flag.StringVar(&conf.dbpass, "db-pass", "changeit", "Database password")
	flag.StringVar(&conf.dbname, "db-name", "mainbase", "Database name")
	flag.StringVar(&conf.lsaddr, "listen-addr", ":8080", "Address to listen, like :8080")
}

func main() {
	var err error

	flag.Parse()

	db, err = gorm.Open("postgres", getConnectionString())
	if err != nil {
		log.Error(err)
		panic("Could not connect to database!")
	}
	autoMigrate()

	// Let it use its Logger() && Recovery() by default
	engine := gin.Default()
	TournamentApi := engine.Group("/tournament/v0")
	mountRoutes(TournamentApi)
	engine.Run(conf.lsaddr)
}

func getConnectionString() string {
	// https://www.postgresql.org/docs/9.5/static/app-postgres.html
	// Let's don't polish a DB connection string forming in this test task
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		&conf.dbhost,
		&conf.dbport,
		&conf.dbuser,
		&conf.dbpass,
		&conf.dbname,
	)
}

func autoMigrate() {
	db.AutoMigrate(
		&User{},
		&UserAuth{},
		&Tournament{},
		&TournamentPlayer{},
		&TournamentBacker{},
		&TournamentWinner{},
		&UserPointsOperations{},
		&UserPointsBalance{},
	)
}

func mountRoutes(api *gin.RouterGroup) {
	api.GET("/info", getTournamentInfo)
	api.GET("/balance", getUserBalance)
	api.POST("/take", takePointsFromUser)
	api.POST("/fund", fundUserWithPoints)
	api.POST("/announceToutnament", announceToutnament)
	api.POST("/joinToutnament", joinTournament)
	api.POST("/resultToutnament", resultTournament)
}
