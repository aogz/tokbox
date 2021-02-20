package tokbox

//Adapted from https://github.com/cioc/tokbox

import (
	"fmt"
	"log"
	"strings"
	"testing"
)

const key = ""
const secret = ""

func TestToken(t *testing.T) {
	tokbox := New(key, secret)
	session, err := tokbox.NewSession("", P2P, ManualArchive)
	if err != nil {
		log.Fatal(err)
		t.FailNow()
	}
	log.Println("Session: ", session)
	hours24 := 24 * 60 * 60
	token, err := session.Token(Publisher, "", int64(hours24))
	if err != nil {
		log.Fatal(err)
		t.FailNow()
	}
	log.Println("Token: ", token)
}

func TestStartArchiving(t *testing.T) {
	tokbox := New(key, secret)
	session, err := tokbox.NewSession("", MediaRouter, ManualArchive)
	if err != nil {
		log.Fatal(err)
		t.FailNow()
	}
	log.Println("Session: ", session)

	_, err2 := session.StartArchiving(true, true)
	if err2 != nil {
		// We should receive 404 here as no clients are connected to the session
		if !strings.Contains(fmt.Sprintln(err2), "404") {
			log.Fatal("Erorr message doesn't contain '404' string")
			t.FailNow()
		}

		if !strings.Contains(fmt.Sprintln(err2), "No clients are actively connected to the OpenTok session.") {
			log.Fatal("Erorr message doesn't contain 'No clients are actively connected to the OpenTok session' string")
			t.FailNow()
		}
	}
}

func TestStopArchiving(t *testing.T) {
	tokbox := New(key, secret)
	session, err := tokbox.NewSession("", MediaRouter, ManualArchive)
	if err != nil {
		log.Fatal(err)
		t.FailNow()
	}
	log.Println("Session: ", session)

	archive := Archive{
		ID: "123-456-789",
		S:  session,
	}

	_, err2 := archive.StopArchiving()
	if err2 != nil {
		// We should receive 404 here as no clients are connected to the session
		if !strings.Contains(fmt.Sprintln(err2), "404") {
			log.Fatal("Erorr message doesn't contain '404' string")
			t.FailNow()
		}

		if !strings.Contains(fmt.Sprintln(err2), "invalid archive ID.") {
			log.Fatal("Erorr message doesn't contain 'No clients are actively connected to the OpenTok session' string")
			t.FailNow()
		}
	}
}
