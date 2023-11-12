package gobard

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Bard struct {
	client         *http.Client
	secure1PSID    string
	secure1PSIDTS  string
	secure1PSIDCC  string
	snlm0e         string
	requestId      int32
	conversationId string
	responseId     string
	choiceId       string
}

var sessionHeaders = map[string]string{
	"Host":          "bard.google.com",
	"X-Same-Domain": "1",
	"User-Agent":    "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36",
	"Content-Type":  "application/x-www-form-urlencoded;charset=UTF-8",
	"Origin":        "https://bard.google.com",
	"Referer":       "https://bard.google.com/",
}

type BardOption func(*Bard)

type Answer struct {
	ConversationId string
	ResponseId     string
	ChoiceId       string
	Content        string
}

type AnswerSummary struct {
	Title     string
	Summaries []string
}

func Secure1PSID(token string) BardOption {
	return func(b *Bard) {
		b.secure1PSID = token
	}
}

func Secure1PSIDTS(token string) BardOption {
	return func(b *Bard) {
		b.secure1PSIDTS = token
	}
}

func Secure1PSIDCC(token string) BardOption {
	return func(b *Bard) {
		b.secure1PSIDCC = token
	}
}

func NewBard(options ...BardOption) *Bard {
	b := &Bard{}

	for _, option := range options {
		option(b)
	}

	b.Init()

	return b
}

func (b *Bard) Init() {
	b.requestId = createRandomRequestId()
	b.client = &http.Client{}
	b.conversationId = ""
	b.responseId = ""
	b.setUpToken()
}

func (b *Bard) setUpToken() {
	snlm0e, err := b.getSnim0e()
	if err != nil {
		panic(err)
	}
	b.snlm0e = snlm0e
}

func (b *Bard) Ask(prompt string) (Answer, error) {
	encodedPrompt := strings.ReplaceAll(strings.ReplaceAll(prompt, "\n", "\\\\n"), "\"", "\\\\\\\"")
	body, query := b.createAskRequest(encodedPrompt)
	plainAnswer, err := b.getAnswer(query, body)
	if err != nil {
		return Answer{}, err
	}

	answer, err := b.parseAnswer(plainAnswer)
	if err != nil {
		return Answer{}, err
	}

	b.updateChainOfThougths(answer)

	return answer, nil
}

func (b *Bard) updateChainOfThougths(answer Answer) {
	b.conversationId = answer.ConversationId
	b.responseId = answer.ResponseId
	b.choiceId = answer.ChoiceId
}

func (b *Bard) createAskRequest(prompt string) (map[string]string, map[string]string) {
	payload := fmt.Sprintf("[null,\"[[\\\"%s\\\",0,null,[],null,null,0],[\\\"ko\\\"],[\\\"%s\\\",\\\"%s\\\",\\\"%s\\\",null,null,[]],null,null,null,[0],0,[],[],1,0]\"]", prompt, b.conversationId, b.responseId, b.choiceId)
	body := map[string]string{
		"f.req": payload,
		"at":    b.snlm0e,
	}

	query := map[string]string{
		"bl":     "boq_assistant-bard-web-server_20231031.09_p4",
		"_reqid": fmt.Sprintf("%d", b.requestId),
		"rt":     "c",
	}

	return body, query
}

func (b *Bard) getAnswer(query map[string]string, body map[string]string) (string, error) {
	content, err := b.PostFormData("https://bard.google.com/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate", query, body)
	if err != nil {
		return "", err
	}

	lines := strings.Split(content, "\n")
	if len(lines) < 4 || lines[3] == "" {
		return "", errors.New("failed to get response")
	}

	return lines[3], nil
}

func (b *Bard) parseAnswer(reponse string) (Answer, error) {
	var root [][]interface{}
	if err := json.Unmarshal([]byte(reponse), &root); err != nil {
		return Answer{}, err
	}
	if len(root) < 1 || len(root[0]) < 3 || root[0][2] == "" {
		return Answer{}, errors.New("failed to get root")
	}

	child := root[0]
	var elements []interface{}
	childContent, ok := child[2].(string)
	if !ok {
		return Answer{}, errors.New("failed to get child content")
	}

	err := json.Unmarshal([]byte(childContent), &elements)
	if err != nil {
		return Answer{}, err
	}
	if len(elements) < 5 {
		return Answer{}, errors.New("failed to get elements")
	}

	id, ok := elements[1].([]interface{})
	if !ok || len(id) < 2 {
		return Answer{}, errors.New("failed to get id")
	}

	contentElements, ok := elements[4].([]interface{})
	if !ok || len(contentElements) < 3 {
		return Answer{}, errors.New("failed to get content elements")
	}

	answerElements, ok := contentElements[2].([]interface{})
	if !ok || len(answerElements) < 2 {
		return Answer{}, errors.New("failed to get answer elements")
	}

	answer := answerElements[1].([]interface{})

	return Answer{
		ConversationId: id[0].(string),
		ResponseId:     id[1].(string),
		ChoiceId:       answerElements[0].(string),
		Content:        answer[0].(string),
	}, nil
}

func (b *Bard) getSnim0e() (string, error) {
	resp, err := b.Get("https://bard.google.com/")
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`SNlM0e":"(.*?)"`)
	matches := re.FindStringSubmatch(resp)
	if len(matches) < 2 {
		return "", errors.New("failed to get SNIM0E")
	}

	return matches[1], nil
}

func (b *Bard) Get(url string) (string, error) {
	if (b.secure1PSIDTS == "") || (b.secure1PSIDCC == "") || (b.secure1PSID == "") {
		panic("token is not set")
	}

	return b.fetch("GET", url, createDefaultHeaders(), b.createDefaultCookie(), nil, nil)
}

func (b *Bard) PostFormData(url string, queryParams map[string]string, bodyParams map[string]string) (string, error) {
	if (b.secure1PSIDTS == "") || (b.secure1PSIDCC == "") || (b.secure1PSID == "") {
		panic("token is not set")
	}

	hds := createDefaultHeaders()
	hds.Set("Content-Type", "application/x-www-form-urlencoded")

	return b.fetch("POST", url, hds, b.createDefaultCookie(), queryParams, createFormReader(bodyParams))
}

func (b *Bard) createDefaultCookie() []http.Cookie {
	return []http.Cookie{
		{
			Name:  "__Secure-1PSID",
			Value: b.secure1PSID,
		},
		{
			Name:  "__Secure-1PSIDTS",
			Value: b.secure1PSIDTS,
		},
		{
			Name:  "__Secure-1PSIDCC",
			Value: b.secure1PSIDCC,
		},
	}
}

func (b *Bard) fetch(method string, url string, hds http.Header, c []http.Cookie, queryParams map[string]string, body io.Reader) (string, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return "", err
	}

	if queryParams != nil {
		q := req.URL.Query()
		for k, v := range queryParams {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	req.Header = hds
	for _, v := range c {
		req.AddCookie(&v)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("failed to fetch")
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func createFormReader(data map[string]string) io.Reader {
	form := url.Values{}
	for k, v := range data {
		form.Add(k, v)
	}
	return strings.NewReader(form.Encode())
}

func createDefaultHeaders() http.Header {
	hds := http.Header{}
	for k, v := range sessionHeaders {
		hds.Set(k, v)
	}

	return hds
}

func createRandomRequestId() int32 {
	var MaxRequestId int32 = 9999
	var MinRequestId int32 = 1000
	return rand.Int31n(MaxRequestId-MinRequestId) + MinRequestId
}
