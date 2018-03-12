showhint () {
  echo "Please provide a command from list [setup|dep|build|install|run|stop|prefill] [options]"
  echo "options for 'build', 'run' and 'stop' are [docker]"
    exit 0
}
showfinstatus () {
  if [ "$1" = 0 ]; then
    echo "Success!"
  else
    echo "Fail!"
  fi
}
if [ -z "$1" ]; then
  showhint
fi
echo Executing $1...
case "$1" in
  setup) export GOPATH=$GOPATH:`pwd`
    echo $GOPATH ;;
  dep) cd src/tournaments
  dep ensure
  cd ../.. ;;
  build)
    if [ "$2" = docker ]; then
      docker build -f Dockerfile .
    else
      rm -rf bin/*
      go build -o bin/tournaments tournaments/...
    fi ;;
  install) go install tournaments/... ;;
  run)
    if [ "$2" = docker ]; then
      docker-compose -f docker-compose.yml -p tournaments up
    else
      docker run --rm -d -e "POSTGRES_USER=postgres" -e "POSTGRES_PASSWORD=changeit" -e "POSTGRES_DB=main" -v pgdata:/var/lib/postgresql/data -p 5432:5432 --hostname postgres --name tournaments-postgres postgres:9.6
      sleep 5
      docker exec -u postgres tournaments-postgres /usr/lib/postgresql/9.6/bin/psql -c "create database main with owner=postgres encoding=utf8;"
      bin/tournaments --listen-addr :8080 --db-host localhost --db-port 5432 --db-user postgres --db-pass changeit --db-name main
    fi ;;
  stop)
      if [ "$2" = docker ]; then
        docker-compose -f docker-compose.yml -p tournaments down
      else
        docker stop tournaments-postgres
      fi ;;
   prefill) docker exec -u postgres tournaments-postgres /usr/lib/postgresql/9.6/bin/psql -d main -c "insert into users (login, password) values('user1', 'pass1'), ('user2', 'pass2'), ('user3', 'pass3');" ;;
  *) showhint ;;
esac
showfinstatus $?
