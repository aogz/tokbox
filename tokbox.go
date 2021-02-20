package tokbox

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"

	"encoding/base64"
	"encoding/json"

	"crypto/hmac"
	"crypto/sha1"

	"fmt"
	"math/rand"
	"strings"
	"time"

	"sync"

	"golang.org/x/net/context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/myesui/uuid"
)

const (
	apiHost              = "https://api.opentok.com"
	apiSession           = "/session/create"
	apiStartArchivingURL = "/v2/project/%s/archive"
	apiStopArchivingURL  = "/v2/project/%s/archive/%s/stop"
)

// MediaMode is the mode of media
type MediaMode string

const (
	// MediaRouter The session will send streams using the OpenTok Media Router.
	MediaRouter MediaMode = "disabled"
	// P2P The session will attempt send streams directly between clients.
	// If clients cannot connect due to firewall restrictions,
	// the session uses the OpenTok TURN server to relay streams.
	P2P = "enabled"
)

// ArchiveMode is the mode of archiving
type ArchiveMode string

const (
	// ManualArchive The session will be manually archived (default option).
	ManualArchive ArchiveMode = "manual"
	// AlwaysArchive The session will be automatically archived.
	AlwaysArchive = "always"
)

// Role is the type of user to be used
type Role string

const (
	// Publisher A publisher can publish streams, subscribe to streams, and signal.
	Publisher Role = "publisher"
	// Subscriber A subscriber can only subscribe to streams.
	Subscriber = "subscriber"
	// Moderator In addition to the privileges granted to a publisher,
	// in clients using the OpenTok.js 2.2 library,
	// a moderator can call the <code>forceUnpublish()</code> and <code>forceDisconnect()</code>
	// method of the Session object.
	Moderator = "moderator"
)

// Tokbox is the main struct to be used for API
type Tokbox struct {
	apiKey        string
	partnerSecret string
	betaURL       string //Endpoint for Beta Programs
}

// Session tokbox session
type Session struct {
	SessionID      string  `json:"session_id"`
	ProjectID      string  `json:"project_id"`
	PartnerID      string  `json:"partner_id"`
	CreateDt       string  `json:"create_dt"`
	SessionStatus  string  `json:"session_status"`
	MediaServerURL string  `json:"media_server_url"`
	T              *Tokbox `json:"-"`
}

// Archive struct represents archive create response
type Archive struct {
	CreatedAt  int      `json:"createdAt"`
	Duration   int      `json:"duration"`
	HasAudio   bool     `json:"hasAudio"`
	HasVideo   bool     `json:"hasVideo"`
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	OutputMode string   `json:"outputMode"`
	ProjectID  int      `json:"projectId"`
	Reason     string   `json:"reason"`
	Resolution string   `json:"resolution"`
	SessionID  string   `json:"sessionId"`
	Size       int      `json:"side"`
	Status     string   `json:"status"`
	URL        string   `json:"url"`
	S          *Session `json:"-"`
}

// New creates a new tokbox instance
func New(apikey, partnerSecret string) *Tokbox {
	return &Tokbox{apikey, partnerSecret, ""}
}

func (t *Tokbox) jwtToken() (string, error) {

	type TokboxClaims struct {
		Ist string `json:"ist,omitempty"`
		jwt.StandardClaims
	}

	claims := TokboxClaims{
		"project",
		jwt.StandardClaims{
			Issuer:    t.apiKey,
			IssuedAt:  time.Now().UTC().Unix(),
			ExpiresAt: time.Now().UTC().Unix() + (2 * 24 * 60 * 60), // 2 hours; //NB: The maximum allowed expiration time range is 5 minutes.
			Id:        uuid.NewV4().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(t.partnerSecret))
}

// NewSession Creates a new tokbox session or returns an error.
// See README file for full documentation: https://github.com/aogz/tokbox
// NOTE: ctx must be nil if *not* using Google App Engine
func (t *Tokbox) NewSession(location string, mm MediaMode, am ArchiveMode, ctx ...context.Context) (*Session, error) {
	params := url.Values{}

	if len(location) > 0 {
		params.Add("location", location)
	}

	params.Add("p2p.preference", string(mm))
	params.Add("archiveMode", string(am))

	var endpoint string
	if t.betaURL == "" {
		endpoint = apiHost
	} else {
		endpoint = t.betaURL
	}
	req, err := http.NewRequest("POST", endpoint+apiSession, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}

	//Create jwt token
	jwt, err := t.jwtToken()
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-OPENTOK-AUTH", jwt)

	if len(ctx) == 0 {
		ctx = append(ctx, nil)
	}
	res, err := client(ctx[0]).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Tokbox returns error code: %v", res.StatusCode)
	}

	var s []Session
	if err = json.NewDecoder(res.Body).Decode(&s); err != nil {
		return nil, err
	}

	if len(s) < 1 {
		return nil, fmt.Errorf("Tokbox did not return a session")
	}

	o := s[0]
	o.T = t
	return &o, nil
}

// StartArchiving starts archiving session
func (s *Session) StartArchiving(archiveVideo bool, archiveAudio bool, ctx ...context.Context) (*Archive, error) {
	var archive Archive

	values := map[string]interface{}{
		"sessionId": s.SessionID,
		"hasAudio":  archiveAudio,
		"hasVideo":  archiveVideo,
	}
	jsonValue, _ := json.Marshal(values)

	url := fmt.Sprintf(apiHost+apiStartArchivingURL, s.T.apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}

	// Create jwt token
	jwt, err := s.T.jwtToken()
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-OPENTOK-AUTH", jwt)

	if len(ctx) == 0 {
		ctx = append(ctx, nil)
	}

	res, err := client(ctx[0]).Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		stringResponse := string(bodyBytes)
		return nil, fmt.Errorf("Tokbox returns error code: %v. Message: %s", res.StatusCode, stringResponse)
	}

	if err = json.NewDecoder(res.Body).Decode(&archive); err != nil {
		return nil, err
	}

	archive.S = s
	return &archive, nil
}

// StopArchiving stops current archive
func (archive *Archive) StopArchiving(ctx ...context.Context) (*Archive, error) {
	var response Archive

	url := fmt.Sprintf(apiHost+apiStopArchivingURL, archive.S.T.apiKey, archive.ID)
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(""))
	if err != nil {
		return nil, err
	}

	// Create jwt token
	jwt, err := archive.S.T.jwtToken()
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-OPENTOK-AUTH", jwt)

	if len(ctx) == 0 {
		ctx = append(ctx, nil)
	}

	res, err := client(ctx[0]).Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		stringResponse := string(bodyBytes)
		return nil, fmt.Errorf("Tokbox returns error code: %v. Message: %s", res.StatusCode, stringResponse)
	}

	if err = json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, err
	}

	response.S = archive.S
	return &response, nil
}

// Token to crate json web token
func (s *Session) Token(role Role, connectionData string, expiration int64) (string, error) {
	now := time.Now().UTC().Unix()

	dataStr := ""
	dataStr += "session_id=" + url.QueryEscape(s.SessionID)
	dataStr += "&create_time=" + url.QueryEscape(fmt.Sprintf("%d", now))
	if expiration > 0 {
		dataStr += "&expire_time=" + url.QueryEscape(fmt.Sprintf("%d", now+expiration))
	}
	if len(role) > 0 {
		dataStr += "&role=" + url.QueryEscape(string(role))
	}
	if len(connectionData) > 0 {
		dataStr += "&connection_data=" + url.QueryEscape(connectionData)
	}
	dataStr += "&nonce=" + url.QueryEscape(fmt.Sprintf("%d", rand.Intn(999999)))

	h := hmac.New(sha1.New, []byte(s.T.partnerSecret))
	n, err := h.Write([]byte(dataStr))
	if err != nil {
		return "", err
	}
	if n != len(dataStr) {
		return "", fmt.Errorf("hmac not enough bytes written %d != %d", n, len(dataStr))
	}

	preCoded := ""
	preCoded += "partner_id=" + s.T.apiKey
	preCoded += "&sig=" + fmt.Sprintf("%x:%s", h.Sum(nil), dataStr)

	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	encoder.Write([]byte(preCoded))
	encoder.Close()
	return fmt.Sprintf("T1==%s", buf.String()), nil
}

// Tokens ...
func (s *Session) Tokens(n int, multithread bool, role Role, connectionData string, expiration int64) []string {
	ret := []string{}

	if multithread {
		var w sync.WaitGroup
		var lock sync.Mutex
		w.Add(n)

		for i := 0; i < n; i++ {
			go func(role Role, connectionData string, expiration int64) {
				a, e := s.Token(role, connectionData, expiration)
				if e == nil {
					lock.Lock()
					ret = append(ret, a)
					lock.Unlock()
				}
				w.Done()
			}(role, connectionData, expiration)

		}

		w.Wait()
		return ret
	}

	for i := 0; i < n; i++ {

		a, e := s.Token(role, connectionData, expiration)
		if e == nil {
			ret = append(ret, a)
		}
	}
	return ret

}
