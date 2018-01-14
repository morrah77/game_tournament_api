#Test task for Go developer

Implement simple game tournament API

##build project

dep ensure

go build

gocker build -f Dockerfile -t main

##Run project

docker-compose up -d
