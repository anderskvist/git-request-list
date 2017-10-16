package gitlab

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/cego/git-request-list/gitrequest"
)

// Client represents a Gitlab merge-request source.
type Client struct {
	http    http.Client
	host    string
	token   string
	verbose bool
	skipWIP bool
}

// repository serves as Unmarshall target type when reading Gitlab API responses.
type repository struct {
	Name string `json:"path_with_namespace"`
	ID   int    `json:"id"`
}

// New produces a new gitlab Client.
func New(host, token string, skipWIP bool, verbose bool) (*Client, error) {
	c := Client{}

	c.http = http.Client{}
	c.host = host
	c.token = token
	c.verbose = verbose
	c.skipWIP = skipWIP

	return &c, nil
}

// GetRequests returns a slice of merge-requests visible to the Client c. If acceptedRepositories is not empty, only
// merge-requests from the repositories whose name is included in acceptedRepositories are returned.
func (c *Client) GetRequests(acceptedRepositories []string) ([]gitrequest.Request, error) {
	whitelist := map[string]bool{}
	for _, repository := range acceptedRepositories {
		whitelist[repository] = true
	}

	var result []gitrequest.Request

	repositories, err := c.getRepositories()
	if err != nil {
		return nil, err
	}

	for _, repository := range repositories {
		if len(whitelist) > 0 && !whitelist[repository.Name] {
			continue
		}

		requests, err := c.getRequests(repository.ID)
		if err != nil {
			return nil, err
		}

		for i := range requests {
			requests[i].RepositoryValue = repository.Name
			result = append(result, &requests[i])
		}
	}

	return result, nil
}

// getRepositories gets the repositories visible to c.
func (c *Client) getRepositories() ([]repository, error) {
	var result []repository

	resp, err := c.get("HEAD", "/projects")
	if err != nil {
		return nil, err
	}
	pageCount, err := strconv.Atoi(resp.Header.Get("X-Total-Pages"))
	if err != nil {
		return nil, err
	}

	for p := 1; p <= pageCount; p++ {
		resp, err = c.get("GET", "/projects?simple=1&with_merge_requests_enabled=1&page="+strconv.Itoa(p))
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		var page []repository
		err = json.NewDecoder(resp.Body).Decode(&page)
		if err != nil {
			return nil, err
		}

		result = append(result, page...)
	}

	return result, nil
}

// getRequests returns all merge-requests of the repository with the given ID visible to c.
func (c *Client) getRequests(repos int) ([]Request, error) {
	var result []Request

	resp, err := c.get("HEAD", "/projects/"+strconv.Itoa(repos)+"/merge_requests?state=opened")
	if err != nil {
		return nil, err
	}
	pageCount, err := strconv.Atoi(resp.Header.Get("X-Total-Pages"))
	if err != nil {
		return nil, err
	}

	for p := 1; p <= pageCount; p++ {
		resp, err := c.get("GET", "/projects/"+strconv.Itoa(repos)+"/merge_requests?state=opened&page="+strconv.Itoa(p))
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		var page []Request

		err = json.NewDecoder(resp.Body).Decode(&page)
		if err != nil {
			return nil, err
		}

		for _, request := range page {
			if c.skipWIP && request.WIP {
				continue
			}

			result = append(result, request)
		}
	}

	return result, nil
}

// get completes a HTTP request to the Gitlab API represented by c.
func (c *Client) get(method string, path string) (*http.Response, error) {
	if c.verbose {
		log.Printf("%s %s/api/v4%s", method, c.host, path)
	}

	req, err := http.NewRequest(method, c.host+"/api/v4"+path, nil)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
