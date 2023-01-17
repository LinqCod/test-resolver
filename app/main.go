package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

const TestBaseURL = "http://185.204.3.165"

var client http.Client

func init() {
	client = http.Client{}
}

func main() {
	r, err := http.Get(TestBaseURL)
	if err != nil {
		log.Fatal(err)
	}
	SID := r.Cookies()[0]

	req, err := http.NewRequest("GET", TestBaseURL+"/question/2", nil)
	if err != nil {
		log.Fatalf("got error: %s", err.Error())
	}
	req.AddCookie(SID)

	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("error occured. Error is: %s", err.Error())
	}
	defer res.Body.Close()

	if _, err = io.Copy(os.Stdout, res.Body); err != nil {
		log.Fatalf("error occured: %s", err.Error())
	}
}
