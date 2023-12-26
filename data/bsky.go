package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	BSKY_XRPC_URI = "https://bsky.social/xrpc/"
)

type sessionResult struct {
	DID          string `json:"did"`
	AccessToken  string `json:"accessJwt"`
	RefreshToken string `json:"refreshJwt"`
}

func createSession() (sessionResult, error) {
	bskyHandle := viper.GetString("server.bsky_handle")
	appPass := viper.GetString("server.bsky_app_pass")

	payload := fmt.Sprintf("{\"identifier\": \"%s\", \"password\": \"%s\"}", bskyHandle, appPass)
	pldReader := bytes.NewReader([]byte(payload))

	res, err := http.Post(BSKY_XRPC_URI+"com.atproto.server.createSession", "application/json", pldReader)
	if err != nil {
		log.Printf("Failed to create BlueSky session: %s", err.Error())
		return sessionResult{}, err
	}
	defer res.Body.Close()

	sRes := sessionResult{}
	if derr := json.NewDecoder(res.Body).Decode(&sRes); derr != nil {
		log.Printf("Failed to decode BlueSky session response: %s", derr.Error())
		return sessionResult{}, err
	}

	return sRes, nil
}

type bskyPost struct {
	Type      string `json:"$type"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
}

type bskyCreatePostPld struct {
	Repo       string   `json:"repo"`
	Collection string   `json:"collection"`
	Record     bskyPost `json:"record"`
}

type bskyCreatePostResp struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

func BskyCreatePost(content string, created time.Time) (string, error) {
	session, err := createSession()
	if err != nil {
		return "", err
	}

	pld := bskyCreatePostPld{
		Repo:       session.DID,
		Collection: "app.bsky.feed.post",
		Record: bskyPost{
			Type:      "app.bsky.feed.post",
			Text:      content,
			CreatedAt: created.Format(time.RFC3339Nano),
		},
	}
	pldJson, err := json.Marshal(pld)
	if err != nil {
		log.Printf("Failed to encode BlueSky create post payload: %s", err.Error())
		return "", err
	}
	log.Println(string(pldJson))
	pldReader := bytes.NewReader(pldJson)

	r, err := http.NewRequest(http.MethodPost, BSKY_XRPC_URI+"com.atproto.repo.createRecord", pldReader)
	if err != nil {
		log.Printf("Failed to create BlueSky post request: %s", err.Error())
		return "", err
	}
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+session.AccessToken)

	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		log.Printf("Failed to create BlueSky post: %s", err.Error())
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("Failed to create BlueSky post, status code: %d", res.StatusCode)
		body, error := ioutil.ReadAll(res.Body)
		if error != nil {
		}
		log.Println(string(body))
		return "", err
	}

	cpRes := bskyCreatePostResp{}
	if derr := json.NewDecoder(res.Body).Decode(&cpRes); derr != nil {
		log.Printf("Failed to decode BlueSky create post response: %s", derr.Error())
		return "", err
	}

	return cpRes.URI, nil
}

type bskyDeletePostPld struct {
	Repo       string `json:"repo"`
	Collection string `json:"collection"`
	RecordKey  string `json:"rkey"`
}

func BskyDeletePost(uri string) error {
	session, err := createSession()
	if err != nil {
		return err
	}

	uriParts := strings.Split(uri, "/")
	rKey := uriParts[len(uriParts)-1]

	pld := bskyDeletePostPld{
		Repo:       session.DID,
		Collection: "app.bsky.feed.post",
		RecordKey:  rKey,
	}
	pldJson, err := json.Marshal(pld)
	if err != nil {
		log.Printf("Failed to encode BlueSky delete post payload: %s", err.Error())
		return err
	}
	pldReader := bytes.NewReader(pldJson)

	r, err := http.NewRequest(http.MethodPost, BSKY_XRPC_URI+"com.atproto.repo.deleteRecord", pldReader)
	if err != nil {
		log.Printf("Failed to delete BlueSky post request: %s", err.Error())
		return err
	}
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+session.AccessToken)

	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		log.Printf("Failed to delete BlueSky post: %s", err.Error())
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("Failed to delete BlueSky post, status code: %d", res.StatusCode)
		return fmt.Errorf("deleteRecord request failed with status code %d", res.StatusCode)
	}

	return nil
}
