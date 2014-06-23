package main

import (
	"errors"
	"fmt"
	"image"
	"log"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
)

const (
	candidatesToRemove = 5
)

// Adds the given file to the cache.
func addToCache(filePath string, img image.Image, format string) error {
	log.Println("Adding to cache:", filePath)

	// Save the image
	size, err := saveImage(img, format, filePath)
	if err == nil {
		key := fmt.Sprintf("image:%s", filePath)

		// Add a record to the cache
		Conn.Do("HSET", key, "size", size)

		Conn.Do("SETNX", "totalcachesize", 0)
		Conn.Do("INCRBY", "totalcachesize", size)

		Conn.Do("ZADD", "imageaccesscounts", 0, key)

		// Update queue of last accesses
		cacheUpdateLastAccess(key)

		pruneCache()
	}

	return err
}

func removeFromCache(key string) {
	size, err := redis.Int(Conn.Do("HGET", key, "size"))
	if err != nil {
		return
	}

	err = deleteImage(strings.Replace(key, "image:", "", 1))
	if err != nil {
		log.Println("Error removing image:", err)
		return
	}

	log.Printf("Removing from cache: %s", key)
	Conn.Do("DEL", key)
	Conn.Do("ZREM", "imageaccesstimestamps", key)
	Conn.Do("ZREM", "imageaccesscounts", key)
	Conn.Do("DECRBY", "totalcachesize", size)
}

// Loads a file specified by its path from the cache.
func loadFromCache(filePath string) (image.Image, string, error) {
	log.Println("Cache lookup for:", filePath)

	exists, err := redis.Bool(Conn.Do("EXISTS", fmt.Sprintf("image:%s", filePath)))
	if err != nil {
		return nil, "", err
	}

	if exists {
		key := fmt.Sprintf("image:%s", filePath)
		cacheUpdateLastAccess(key)

		return loadImage(filePath)
	}

	return nil, "", errors.New("image not found")
}

func cacheUpdateLastAccess(key string) {
	timestamp := time.Now().Unix()
	Conn.Do("ZADD", "imageaccesstimestamps", timestamp, key)
	Conn.Do("ZINCRBY", "imageaccesscounts", 1, key)
}

func pruneCache() {
	go func() {
		if Config.cacheLimit == 0 {
			return
		}

		totalCacheSize, err := redis.Int(Conn.Do("GET", "totalcachesize"))
		if err != nil {
			return
		}

		if totalCacheSize < Config.cacheLimit {
			return
		}

		candidates := getCacheRemovalCandidates()
		for _, candidate := range candidates {
			removeFromCache(candidate)
		}
	}()
}

func getCacheRemovalCandidates() []string {
	set := "imageaccesstimestamps" // LRU
	if Config.cacheStrategy == LFU {
		set = "imageaccesscounts"
	}
	// Remove multiple for better performance (especially LFU)
	candidates, err := redis.Strings(Conn.Do("ZRANGE", set, 0, candidatesToRemove-1))
	if err == nil && len(candidates) > 0 {
		return candidates
	}
	return nil
}
