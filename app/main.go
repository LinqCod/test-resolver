package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	TestBaseURL = "http://185.204.3.165"

	RegexRadioInput  = "(\\>{0,1})<input type=\"radio\" name=\"(\\w*)\" value=\"(\\w*)\">"
	RegexTextInput   = "<input type=\"text\" name=\"(\\w*)\">"
	RegexSelectInput = "<select name=\"(\\w*)\">|<option value=\"(\\w*)\">\\w*<\\/option>"
)

var client http.Client
var radioRegex *regexp.Regexp
var textRegex *regexp.Regexp
var selectRegex *regexp.Regexp

func init() {
	client = http.Client{}

	radioRegex = regexp.MustCompile(RegexRadioInput)
	textRegex = regexp.MustCompile(RegexTextInput)
	selectRegex = regexp.MustCompile(RegexSelectInput)
}

func main() {
	r, err := http.Get(TestBaseURL)
	if err != nil {
		log.Fatalf("error while visiting test home page: %s", err.Error())
	}

	SID := r.Cookies()[0]
	solveAllQuestions(SID)
}

func solveAllQuestions(SID *http.Cookie) {
	index := 1
	for true {
		isTestCompleted := solveQuestion(index, SID)
		if isTestCompleted {
			break
		}
		index++
	}
}

func solveQuestion(index int, SID *http.Cookie) bool {
	req, err := http.NewRequest("GET", TestBaseURL+"/question/"+strconv.Itoa(index), nil)
	if err != nil {
		log.Fatalf("got error: %s", err.Error())
	}
	req.AddCookie(SID)

	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("error occured. Error is: %s", err.Error())
	}
	defer res.Body.Close()

	htmlBytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("error while reading response body: %s", err.Error())
	}
	html := string(htmlBytes)

	textResult := textRegex.FindAllStringSubmatch(html, -1)
	radioResult := radioRegex.FindAllStringSubmatch(html, -1)
	selectResult := selectRegex.FindAllStringSubmatch(html, -1)

	postBody := make(map[string]string)
	if len(textResult) > 0 {
		for _, v := range textResult {
			postBody[v[1]] = "test"
		}
	}
	if len(radioResult) > 0 {
		currentRadioGroupName := radioResult[0][2]
		currentRadioGroupMaxLengthValue := radioResult[0][3]
		currentRadioGroupMaxLength := len(currentRadioGroupMaxLengthValue)
		for i := 1; i < len(radioResult); i++ {
			if radioResult[i][1] != "" {
				postBody[currentRadioGroupName] = currentRadioGroupMaxLengthValue
				currentRadioGroupName = radioResult[i][2]
				currentRadioGroupMaxLengthValue = radioResult[i][3]
				currentRadioGroupMaxLength = len(currentRadioGroupMaxLengthValue)
				continue
			}
			if currentRadioGroupMaxLength < len(radioResult[i][3]) {
				currentRadioGroupMaxLengthValue = radioResult[i][3]
				currentRadioGroupMaxLength = len(currentRadioGroupMaxLengthValue)
			}
		}
		postBody[currentRadioGroupName] = currentRadioGroupMaxLengthValue
	}
	if len(selectResult) > 0 {
		currentSelectGroupName := selectResult[0][1]
		currentSelectGroupMaxLengthValue := selectResult[1][2]
		currentSelectGroupMaxLength := len(currentSelectGroupMaxLengthValue)
		for i := 2; i < len(selectResult); i++ {
			if selectResult[i][1] != "" {
				postBody[currentSelectGroupName] = currentSelectGroupMaxLengthValue
				currentSelectGroupName = selectResult[i][1]
				currentSelectGroupMaxLengthValue = ""
				currentSelectGroupMaxLength = 0
				continue
			}
			if currentSelectGroupMaxLength < len(selectResult[i][2]) {
				currentSelectGroupMaxLengthValue = selectResult[i][2]
				currentSelectGroupMaxLength = len(currentSelectGroupMaxLengthValue)
			}
		}
		postBody[currentSelectGroupName] = currentSelectGroupMaxLengthValue
	}

	dataToSend := url.Values{}
	for k, v := range postBody {
		dataToSend.Add(k, v)
	}

	result := postData(index, SID, dataToSend)
	return strings.Contains(result, "Test successfully passed")
}

func postData(index int, SID *http.Cookie, data url.Values) string {
	req, err := http.NewRequest("POST", TestBaseURL+"/question/"+strconv.Itoa(index), strings.NewReader(data.Encode()))
	if err != nil {
		log.Fatalf("got error: %s", err.Error())
	}
	req.AddCookie(SID)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("error occured. Error is: %s", err.Error())
	}
	defer res.Body.Close()

	htmlBytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("error while reading response body: %s", err.Error())
	}

	html := string(htmlBytes)
	fmt.Println(html)
	return html
}
