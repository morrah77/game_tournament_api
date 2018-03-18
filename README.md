#Test task for Go developer

Implement simple game tournament API

##About structure
to have an ability to store some infrastructure entities inside of project directory, project contains its own `src` directory.

Due to simplest implementation (GORM + gin under the hood) contains just one package `main`.

For conveniency contains main control script (POSIX sh only, not for Win-like OSes) supporting a few important commands.

##build project

`./control.sh dep && ./control.sh build`

with docker:

`./control.sh dep && ./control.sh build docker`

or manually:

`cd src/tournaments && dep ensure && cd ../../`

`go build -o bin/tournaments tournaments/...`

with docker:

`docker build -f Dockerfile -t tournaments .`

##Run project

locally (postgresql DB runs in docker container)

`./control.sh run`

with docker:

`./control.sh run docker`

or

`docker-compose -p tournaments up -d`

#Stop project

if runs locally

`./control.sh stop`

if runs with docker:

`./control.sh stop docker`

or

`docker-compose -p tournaments down`

##Test

Due to simple implementation I didn't found a place for common unit tests (it seems rather inconvenient to test such way executable package), just a manual testing is possible for now.

Probably `main` package should be divided to main && auxiliary ones, so, it'd be a good point to cover aux. package by unit tests.

Of course, it's possible to automate end-to-end testing using any appropriate test framework like Geb (it's on Groovy, but for black-box testing via network it doesn't mind) or even make a bash script calling curl commands && matching responses to expectations, but it seems being out of this task bounds.

###Manually

Before start manual testing please prefill DB by some users:

`./control.sh  prefill`

Then it's possible to play with users' balances, tournaments && results with requests like following:

`curl -iv -X GET http://localhost:8080/tournament/v0/user/balance?id=1`

`curl -iv -X POST http://localhost:8080/tournament/v0/user/fund -d '{"player_id":1,"points":100}' -H "Content-Type:application/json"`

`curl -iv -X POST http://localhost:8080/tournament/v0/user/take -d '{"player_id":1,"points":100}' -H "Content-Type:application/json"`


`curl -iv -X POST http://localhost:8080/tournament/v0/tournament/announceTournament -d '{"date":"2018-03-18T00:59:00Z","deposit":200,"game_id":1}' -H "Content-Type:application/json"`

`curl -iv -X GET http://localhost:8080/tournament/v0/tournament/list?limit=20\&offset=0`

`curl -iv -X GET http://localhost:8080/tournament/v0/tournament/info?id=1`

`curl -iv -X POST http://localhost:8080/tournament/v0/tournament/joinTournament -d '{"tournament_id":1,"player_id":1, "backer_ids":[2,3]}' -H "Content-Type:application/json"`

`curl -iv -X POST http://localhost:8080/tournament/v0/tournament/resultTournament -d '{"tournament_id":1,"winners":[{"player_id":1,"prize":500}]}' -H "Content-Type:application/json"`

####Manual test

#####Fund users with balances

`curl -iv -X POST http://localhost:8080/tournament/v0/user/fund -d '{"player_id":1,"points":300}' -H "Content-Type:application/json"`

`curl -iv -X POST http://localhost:8080/tournament/v0/user/fund -d '{"player_id":2,"points":300}' -H "Content-Type:application/json"`

`curl -iv -X POST http://localhost:8080/tournament/v0/user/fund -d '{"player_id":3,"points":300}' -H "Content-Type:application/json"`

`curl -iv -X POST http://localhost:8080/tournament/v0/user/fund -d '{"player_id":4,"points":500}' -H "Content-Type:application/json"`

`curl -iv -X POST http://localhost:8080/tournament/v0/user/fund -d '{"player_id":5,"points":1000}' -H "Content-Type:application/json"`

#####Announce tournament with 1000 points deposit
`curl -iv -X POST http://localhost:8080/tournament/v0/tournament/announceTournament -d '{"date":"2018-03-19T00:59:00Z","deposit":1000,"game_id":1}' -H "Content-Type:application/json"`

#####User#5 joins tournament on his own

`curl -iv -X POST http://localhost:8080/tournament/v0/tournament/joinTournament -d '{"tournament_id":1,"player_id":5}' -H "Content-Type:application/json"`

#####User#1 joins tournament backed by users #2, #3, #4

`curl -iv -X POST http://localhost:8080/tournament/v0/tournament/joinTournament -d '{"tournament_id":1,"player_id":1, "backer_ids":[2,3,4]}' -H "Content-Type:application/json"`

#####User#1 wins tournament with 2000 points prize

`curl -iv -X POST http://localhost:8080/tournament/v0/tournament/resultTournament -d '{"tournament_id":1,"winners":[{"player_id":1,"prize":2000}]}' -H "Content-Type:application/json"`

#####Usser' balances check

`curl -iv -X GET http://localhost:8080/tournament/v0/user/balance?id=1`

Users #1,  #2, #3: 550 points expected

`curl -iv -X GET http://localhost:8080/tournament/v0/user/balance?id=2`

`curl -iv -X GET http://localhost:8080/tournament/v0/user/balance?id=3`

`curl -iv -X GET http://localhost:8080/tournament/v0/user/balance?id=4`

User #4: 750 points expected

`curl -iv -X GET http://localhost:8080/tournament/v0/user/balance?id=5`

User #5: 0 points expected
