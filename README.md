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

curl -iv http://localhost:8080/tournament/v0/announceTournament -X POST -d '{"date":"2018-03-13T02:59:00Z","deposit":200,"game_id":1}' -H "Content-Type:application/json"

curl -iv http://localhost:8080/tournament/v0/info?id=1 -X GET

curl -iv http://localhost:8080/tournament/v0/balance?id=1 -X GET

curl -iv http://localhost:8080/tournament/v0/fund -X POST -d '"playerId":1,"points":100' -H "Content-Type:application/json"

curl -iv http://localhost:8080/tournament/v0/take -X POST -d '"playerId":1,"points":100' -H "Content-Type:application/json"

curl -iv http://localhost:8080/tournament/v0/joinTournament -X POST -d '{"tournamentId"1,"playerId":1, "backerIds":[2,3,4]}' -H "Content-Type:application/json"

curl -iv http://localhost:8080/tournament/v0/resultTournament -X POST -d '{"tournamentId":1,"winners":[{"playerId":1,"prize":500}]}' -H "Content-Type:application/json"

