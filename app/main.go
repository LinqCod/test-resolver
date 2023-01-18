package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
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

func worker(id int, wg *sync.WaitGroup) {
	defer wg.Done()

	if err := solveTest(); err != nil {
		log.Fatalf("error while solving test with worker: %d", id)
	}

	fmt.Printf("worker: %d, test completed!\n", id)
}

func main() {
	workersCount := 1
	var err error
	if len(os.Args) > 1 {
		workersCount, err = strconv.Atoi(os.Args[1])
		if err != nil {
			log.Fatalf("error while parsing run args: %s", err.Error())
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < workersCount; i++ {
		wg.Add(1)
		go worker(i, &wg)
	}

	wg.Wait()
}

func solveTest() error {
	r, err := http.Get(TestBaseURL)
	if err != nil {
		return errors.New(fmt.Sprintf("error while visiting test home page: %s", err.Error()))
	}

	SID := r.Cookies()[0]

	index := 1
	for true {
		questionAnswers, err := solveQuestionByIndex(index, SID)
		if err != nil {
			return err
		}
		isTestCompleted, err := postAnswerForQuestionByIndex(index, SID, questionAnswers)
		if err != nil {
			return err
		}
		if isTestCompleted {
			break
		}
		index++
	}

	return nil
}

func solveQuestionByIndex(index int, SID *http.Cookie) (url.Values, error) {
	req, err := http.NewRequest("GET", TestBaseURL+"/question/"+strconv.Itoa(index), nil)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error while creating new get request: %s", err.Error()))
	}
	req.AddCookie(SID)

	res, err := client.Do(req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error while doing get request: %s", err.Error()))
	}
	defer res.Body.Close()

	htmlBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error while reading response body: %s", err.Error()))
	}
	html := string(htmlBytes)

	textResult := textRegex.FindAllStringSubmatch(html, -1)
	radioResult := radioRegex.FindAllStringSubmatch(html, -1)
	selectResult := selectRegex.FindAllStringSubmatch(html, -1)

	postBody := make(map[string]string)

	// solving text inputs
	if len(textResult) > 0 {
		for _, v := range textResult {
			postBody[v[1]] = "test"
		}
	}
	// solving radio inputs
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
	// solving select inputs
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

	resultData := url.Values{}
	for k, v := range postBody {
		resultData.Add(k, v)
	}

	return resultData, nil
}

func postAnswerForQuestionByIndex(index int, SID *http.Cookie, data url.Values) (bool, error) {
	req, err := http.NewRequest("POST", TestBaseURL+"/question/"+strconv.Itoa(index), strings.NewReader(data.Encode()))
	if err != nil {
		return false, errors.New(fmt.Sprintf("error while creating post request: %s", err.Error()))
	}
	req.AddCookie(SID)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		return false, errors.New(fmt.Sprintf("error while doing post request: %s", err.Error()))
	}
	defer res.Body.Close()

	htmlBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return false, errors.New(fmt.Sprintf("error while reading response body: %s", err.Error()))
	}

	html := string(htmlBytes)
	return strings.Contains(html, "Test successfully passed"), nil
}
