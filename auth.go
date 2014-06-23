package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"github.com/garyburd/redigo/redis"
	"github.com/twinj/uuid"
)

const (
	// GetPermission = permission to get (and transform) images
	GetPermission = "get"
	// UploadPermission = permission to upload images
	UploadPermission = "upload"
)

var (
	permissionsByKey map[string]map[string]bool
)

func init() {
	// Change the UUID format to remove surrounding braces and dashes
	uuid.SwitchFormat(uuid.Clean, true)
}

func authInit() error {
	keys, err := listKeys()
	if err != nil {
		return err
	}

	permissionsByKey = make(map[string]map[string]bool)

	// Set up permissions for when there's no API key
	permissionsByKey[""] = make(map[string]bool)
	permissionsByKey[""][GetPermission] = !Config.authorisedGet
	permissionsByKey[""][UploadPermission] = !Config.authorisedUpload

	// Set up permissions for API keys
	for _, key := range keys {
		permissions, err := infoAboutKey(key)
		if err != nil {
			return err
		}
		permissionsByKey[key] = make(map[string]bool)
		for _, permission := range permissions {
			permissionsByKey[key][permission] = true
		}
	}

	return nil
}

func hasPermission(key, permission string) bool {
	val, ok := permissionsByKey[key][permission]
	if ok {
		return val
	}
	return false
}

func generateKey() (string, string, error) {
	key := uuid.NewV4().String()
	secretKey := uuid.NewV4().String()
	_, err := Conn.Do("SADD", "api-keys", key)
	if err != nil {
		return "", "", err
	}
	_, err = Conn.Do("HSET", "key:"+key, "secret", secretKey)
	if err != nil {
		return "", "", err
	}
	_, err = Conn.Do("SADD", "key:"+key+":permissions", GetPermission, UploadPermission)
	return key, secretKey, err
}

func generateSecret(key string) (string, error) {
	err := checkKeyExists(key)
	if err != nil {
		return "", err
	}

	secretKey := uuid.NewV4().String()
	_, err = Conn.Do("HSET", "key:"+key, "secret", secretKey)
	if err != nil {
		return "", err
	}

	return secretKey, nil
}

func infoAboutKey(key string) ([]string, error) {
	err := checkKeyExists(key)
	if err != nil {
		return nil, err
	}
	permissions, err := redis.Strings(Conn.Do("SMEMBERS", "key:"+key+":permissions"))
	if err != nil {
		return nil, err
	}
	sort.Strings(permissions)
	return permissions, nil
}

func listKeys() ([]string, error) {
	return redis.Strings(Conn.Do("SMEMBERS", "api-keys"))
}

func modifyKey(key, op, permission string) error {
	err := checkKeyExists(key)
	if err != nil {
		return err
	}
	if op != "add" && op != "remove" {
		return errors.New("modifier needs to be 'add' or 'remove'")
	}
	if permission != GetPermission && permission != UploadPermission {
		return fmt.Errorf("modifier needs to end with a valid permission: %s or %s", GetPermission, UploadPermission)
	}
	if op == "add" {
		_, err = Conn.Do("SADD", "key:"+key+":permissions", permission)
	} else {
		_, err = Conn.Do("SREM", "key:"+key+":permissions", permission)
	}
	return err
}

func removeKey(key string) error {
	err := checkKeyExists(key)
	if err != nil {
		return err
	}
	_, err = Conn.Do("SREM", "api-keys", key)
	if err != nil {
		return err
	}
	_, err = Conn.Do("DEL", "key:"+key+":permissions")
	return err
}

func getSecretForKey(key string) (string, error) {
	err := checkKeyExists(key)
	if err != nil {
		return "", err
	}

	secret, err := redis.String(Conn.Do("HGET", "key:"+key, "secret"))
	if err != nil {
		return "", err
	}

	return secret, nil
}

func authPermissionsOptions() string {
	return fmt.Sprintf("%s/%s", GetPermission, UploadPermission)
}

func checkKeyExists(key string) error {
	exists, err := redis.Bool(Conn.Do("SISMEMBER", "api-keys", key))
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("key does not exist")
	}
	return nil
}

func isValidSignature(signature, secret string, queryParams map[string]string) bool {
	var keys []string
	for key := range queryParams {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	queryString := ""
	for _, key := range keys {
		if queryString != "" {
			queryString += "&"
		}
		queryString += key + "=" + queryParams[key]
	}

	expected := signQueryString(queryString, secret)
	decodedSignature, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return hmac.Equal(decodedSignature, expected)
}

func signQueryString(queryString, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(queryString))
	return mac.Sum(nil)
}
