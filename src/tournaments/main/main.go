// Copyright 2018 h.lazar. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
/*
	Implements simple  social tournaments service with REST API
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"log"

	"github.com/morrah77/go-developer-test-task-2/src/tournaments/api"
	"github.com/morrah77/go-developer-test-task-2/src/tournaments/storage"
)

const LOG_PREFIX = `Tournaments `

var (
	logger  *log.Logger
	dbConf  *storage.DsnColfig
	apiConf *api.ApiConf
)

func init() {
	dbConf = &storage.DsnColfig{}
	apiConf = &api.ApiConf{}
	flag.StringVar(&dbConf.DbHost, "db-host", "postgres", "Database host")
	flag.StringVar(&dbConf.DbPort, "db-port", "5432", "Database port")
	flag.StringVar(&dbConf.DbUser, "db-user", "postgres", "Database username")
	flag.StringVar(&dbConf.DbPass, "db-pass", "changeit", "Database password")
	flag.StringVar(&dbConf.DbName, "db-name", "main", "Database name")
	flag.StringVar(&apiConf.ListenAddr, "listen-addr", ":8080", "Address to listen, like :8080")
	flag.StringVar(&apiConf.RelativePath, "api-path", "/tournament/v0", "Api path, like /tournament/v0")

	logger = log.New(os.Stdout, LOG_PREFIX, log.Flags())
}

func main() {
	var (
		//stopChan           chan os.Signal
		err            error
		stor           interface{}
		tournamentsApi *api.Api
	)

	defer func() {
		fmt.Printf("Deferred cleanup\n")
		if stor != nil {
			if cs, ok := stor.(io.Closer); ok {
				err = cs.Close()
			}
			if err != nil {
				logger.Print(err.Error())
			}
		}
	}()

	//stopChan = make(chan os.Signal, 1)
	//signal.Notify(stopChan)

	flag.Parse()

	if stor, err = storage.NewStorage(dbConf, logger); err != nil {
		panic(err.Error())
	}

	tournamentsApi, err = api.NewApi(apiConf, stor, logger)
	if err != nil {
		panic(err.Error())
	}
	if err = tournamentsApi.Run(); err != nil {
		panic(err.Error())
	}

	//s := <-stopChan
	//fmt.Printf("OS signal received: %#v\n", s)
	return
}
