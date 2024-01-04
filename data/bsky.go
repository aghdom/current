package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
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

type bskyFacetIndex struct {
	ByteStart int `json:"byteStart"`
	ByteEnd   int `json:"byteEnd"`
}

type bskyFacetFeature struct {
	Type string `json:"$type"`
	DID  string `json:"did,omitempty"`
	URI  string `json:"uri,omitempty"`
	Tag  string `json:"tag,omitempty"`
}

type bskyFacet struct {
	Index    bskyFacetIndex     `json:"index"`
	Features []bskyFacetFeature `json:"features"`
}

type bskyPost struct {
	Type      string      `json:"$type"`
	Text      string      `json:"text"`
	CreatedAt string      `json:"createdAt"`
	Facets    []bskyFacet `json:"facets,omitempty"`
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

func trimEmphasis(content string) string {
	// Remove Bold
	rb := regexp.MustCompile(`\*\*([\w\s]*)\*\*`)
	result := rb.ReplaceAll([]byte(content), []byte("$1"))

	// Remove Italics
	ri := regexp.MustCompile(`\*([\w\s]*)\*`)
	result = ri.ReplaceAll(result, []byte("$1"))

	// Remove Strikethrough
	rs := regexp.MustCompile(`~~([\w\s]*)~~`)
	result = rs.ReplaceAll(result, []byte("$1"))

	// Remove Code
	rc := regexp.MustCompile("`([^`]*)`")
	result = rc.ReplaceAll(result, []byte("$1"))

	return string(result)
}

func bskyResolveHandle(handle string) string {
	res, err := http.Get(BSKY_XRPC_URI + "com.atproto.identity.resolveHandle?handle=" + handle)
	if err != nil || res.StatusCode == http.StatusBadRequest {
		//If we fail to resolve, just continue
		return ""
	}
	defer res.Body.Close()
	var data map[string]string
	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return ""
	}
	return data["did"]
}

func parseMentions(content string) []bskyFacet {
	var mentions []bskyFacet
	mre := regexp.MustCompile(`[$|\W](@([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)`)
	locs := mre.FindAllSubmatchIndex([]byte(content), -1)
	for _, loc := range locs {
		handle := string(content[loc[2]+1 : loc[3]]) // +1 to skip the initial @ symbol
		did := bskyResolveHandle(handle)
		if did == "" {
			// We failed to parse handle, skip this mention, it will be shown as plain text
			continue
		}
		m := bskyFacet{
			Index: bskyFacetIndex{
				ByteStart: loc[0],
				ByteEnd:   loc[1],
			},
			Features: []bskyFacetFeature{{
				Type: "app.bsky.richtext.facet#mention",
				DID:  did,
			}},
		}
		mentions = append(mentions, m)
	}

	return mentions
}

func parseLinks(content string) (string, []bskyFacet) {
	var links []bskyFacet
	re := regexp.MustCompile(`\[(?P<text>.+)\]\((?P<target>.+)\)`)
	for {
		loc := re.FindStringSubmatchIndex(content)
		if loc == nil {
			break
		}
		start := loc[0]
		fullMatch := string(content[loc[0]:loc[1]])
		text := string(content[loc[2]:loc[3]])
		target := string(content[loc[4]:loc[5]])

		content = strings.Replace(content, fullMatch, text, 1)
		links = append(links, bskyFacet{
			Index: bskyFacetIndex{
				ByteStart: start,
				ByteEnd:   start + len(text),
			},
			Features: []bskyFacetFeature{{
				Type: "app.bsky.richtext.facet#link",
				URI:  target,
			}},
		})
	}
	return content, links
}

func parseFacets(content string) (string, []bskyFacet) {
	var facets []bskyFacet
	//Links must be parsed first, as they change the content
	content, links := parseLinks(content)
	facets = append(facets, links...)
	facets = append(facets, parseMentions(content)...)

	return content, facets
}

func convertToBskyPost(content string, created time.Time) bskyPost {
	content = trimEmphasis(content)
	content, facets := parseFacets(content)
	p := bskyPost{
		Type:      "app.bsky.feed.post",
		Text:      content,
		CreatedAt: created.Format(time.RFC3339Nano),
		Facets:    facets,
	}
	return p
}

func BskyCreatePost(content string, created time.Time) (string, error) {
	session, err := createSession()
	if err != nil {
		return "", err
	}

	pld := bskyCreatePostPld{
		Repo:       session.DID,
		Collection: "app.bsky.feed.post",
		Record:     convertToBskyPost(content, created),
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
