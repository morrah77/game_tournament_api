package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

const LOG_PREFIX = `Tournaments `

var (
	db     *gorm.DB
	logger *log.Logger
)

var conf struct {
	dbhost string
	dbport string
	dbuser string
	dbpass string
	dbname string
	lsaddr string
}

func init() {
	flag.StringVar(&conf.dbhost, "db-host", "postgres", "Database host")
	flag.StringVar(&conf.dbport, "db-port", "5432", "Database port")
	flag.StringVar(&conf.dbuser, "db-user", "postgres", "Database username")
	flag.StringVar(&conf.dbpass, "db-pass", "changeit", "Database password")
	flag.StringVar(&conf.dbname, "db-name", "mainbase", "Database name")
	flag.StringVar(&conf.lsaddr, "listen-addr", ":8080", "Address to listen, like :8080")

	logger = log.New(os.Stdout, LOG_PREFIX, log.Flags())
}

func main() {
	var (
		err                error
		dbConnectionString string
		connectionAttempts int
	)

	defer func() {
		if db != nil {
			err = db.Close()
		}
		if err != nil {
			logger.Print(err.Error())
		}
	}()

	flag.Parse()

	dbConnectionString = getConnectionString()

	for {
		db, err = gorm.Open("postgres", dbConnectionString)
		if err == nil {
			logger.Print(`DB Connection success!`)
			break
		}
		logger.Print(err.Error())
		connectionAttempts++
		if connectionAttempts > 10 {
			panic("Could not connect to database!")
		}
		time.Sleep(5 * time.Second)
	}

	autoMigrate()

	// Let it use its Logger() && Recovery() by default
	engine := gin.Default()
	TournamentApi := engine.Group("/tournament/v0")
	mountRoutes(TournamentApi)
	err = engine.Run(conf.lsaddr)
	if err != nil {
		logger.Print(err.Error())
	}
}

func getConnectionString() string {
	// https://www.postgresql.org/docs/9.5/static/app-postgres.html
	return fmt.Sprintf("host=%#s port=%#s user=%#s password=%#s dbname=%#s sslmode=disable",
		conf.dbhost,
		conf.dbport,
		conf.dbuser,
		conf.dbpass,
		conf.dbname,
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
	api.POST("/announceTournament", announceTournament)
	api.POST("/joinTournament", joinTournament)
	api.POST("/resultTournament", resultTournament)
}
