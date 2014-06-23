package main

import (
	"os"
	"strconv"

	"github.com/garyburd/redigo/redis"
	"github.com/soveran/redisurl"
)

const (
	redisPortEnvVar  = "PIXLSERV_REDIS_PORT"
	redisURLEnvVar   = "PIXLSERV_REDIS_URL"
	redisDefaultPort = 6379
)

var (
	// Conn is a global redis connection object
	Conn redis.Conn
)

func redisInit() error {
	url := os.Getenv(redisURLEnvVar)
	var err error
	if url != "" {
		Conn, err = redisurl.ConnectToURL(url)
	} else {
		port, errLocal := strconv.Atoi(os.Getenv(redisPortEnvVar))
		if errLocal != nil {
			port = redisDefaultPort
		}

		Conn, errLocal = redis.Dial("tcp", ":"+strconv.Itoa(port))
		err = errLocal
	}
	return err
}

func redisCleanUp() {
	Conn.Close()
}
