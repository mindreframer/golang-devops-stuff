// +build gotask

package examples

import (
	"encoding/json"
	"fmt"
	"github.com/jingweno/gotask/tasking"
	"io/ioutil"
	"net/http"
)

// NAME
//    gh-user - Get URL for a given GitHub user login
//
// DESCRIPTION
//    Given a GitHub user login, call the GitHub API to get this user and print out the user page URL.
//
//    For example
//
//    $ gotask git-hub-user jingweno
//
// OPTIONS
//    --verbose, -v
//        run in verbose mode
func TaskGitHubUser(t *tasking.T) {
	if len(t.Args) == 0 {
		t.Error("No GitHub user login is provided!")
		return
	}

	login := t.Args[0]
	data, err := fetchGitHubUser(login)
	if err != nil {
		t.Error(err)
		return
	}

	url, ok := data["html_url"]
	if !ok {
		t.Errorf("No URL found for user login %s\n", login)
		return
	}

	t.Logf("The URL for user %s is %s\n", login, url)
}

func fetchGitHubUser(login string) (data map[string]interface{}, err error) {
	url := fmt.Sprintf("https://api.github.com/users/%s", login)
	res, err := http.Get(url)
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	err = json.Unmarshal(body, &data)
	return
}
