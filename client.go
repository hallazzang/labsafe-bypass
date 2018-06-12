package labsafe

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

const UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/66.0.3359.181 Safari/537.36"

const (
	Unknown = iota
	NormalContent
	VideoContent
)

type Client struct {
	hc *http.Client
}

type Progress struct {
	Type  int
	Name  string
	No    string
	Taken bool
}

func NewClient() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create new cookiejar")
	}

	return &Client{
		hc: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Jar: jar,
		},
	}, nil
}

func (c *Client) Login(id, pw string) (bool, error) {
	u := "https://labsafe.pknu.ac.kr/Account/AjxAgreementChk"
	data := url.Values{
		"AgencyNo":  {"100"},
		"LoginType": {"1"},
		"UniqueKey": {id},
		"Password":  {pw},
	}
	req, err := http.NewRequest("POST", u, strings.NewReader(data.Encode()))
	if err != nil {
		return false, errors.Wrap(err, "failed creating new http.Request")
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Add("Referer", "https://labsafe.pknu.ac.kr/Account/LogOn")
	req.Header.Add("User-Agent", UserAgent)
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	resp, err := c.hc.Do(req)
	if err != nil {
		return false, errors.Wrap(err, "failed sending http request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, errors.Errorf("bad status code: %d", resp.StatusCode)
	}

	var r struct {
		IsSuccess bool
		// IsAgree   bool
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return false, errors.Wrap(err, "json decode error")
	}

	return r.IsSuccess, nil
}

func (c *Client) GetTotalPages(progressNo string) (int, error) {
	u := "https://labsafe.pknu.ac.kr/Edu/ContentsViewPop"
	data := url.Values{
		"scheduleMemberProgressNo": {progressNo},
	}
	resp, err := c.hc.Get(fmt.Sprintf("%s?%s", u, data.Encode()))
	if err != nil {
		return 0, errors.Wrap(err, "http get error")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, errors.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, errors.Wrap(err, "failed reading response body")
	}

	m := searchRegexp(`var totalPage = '(\d+)';`, string(body))
	if len(m) == 0 {
		return 0, errors.New("failed finding matching text")
	}
	tp, _ := strconv.Atoi(m[1])

	return tp, nil
}

func (c *Client) MemberNo() (string, error) {
	u := "https://labsafe.pknu.ac.kr/Edu/OnLineEdu"
	resp, err := c.hc.Get(u)
	if err != nil {
		return "", errors.Wrap(err, "http get error")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed reading response body")
	}

	m := searchRegexp(`var m_ScheduleMemberNo = "(\d+)";`, string(body))
	if len(m) == 0 {
		return "", errors.New("failed finding matching text")
	}

	return m[1], nil
}

func (c *Client) GetProgresses() ([]Progress, error) {
	no, err := c.MemberNo()
	if err != nil {
		return nil, errors.Wrap(err, "failed getting member no")
	}

	u := "https://labsafe.pknu.ac.kr/Edu/ProgressInfoList"
	data := url.Values{
		"scheduleMemberNo": {no},
	}
	resp, err := c.hc.PostForm(u, data)
	if err != nil {
		return nil, errors.Wrap(err, "http post error")
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating document from response")
	}

	var p []Progress
	var innerError error
	doc.Find(".edufireTable tr:not(.edufireTableTop)").EachWithBreak(func(i int, s *goquery.Selection) bool {
		typ := NormalContent
		name := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
		no := ""
		taken := false

		t := s.Find("td:nth-child(7) input").First()
		if t.Size() > 0 {
			onclick, exists := t.Attr("onclick")
			if !exists {
				innerError = errors.New("'onclick' attribute does not exist")
				return false
			}
			m := searchRegexp(`OpenContentViewPop\((\d+)\)`, onclick)
			if len(m) == 0 {
				m = searchRegexp(`OpenContentViewPopAvi\((\d+)\)`, onclick)
				if len(m) == 0 {
					innerError = errors.New("failed finding matching text")
					return false
				}
				typ = VideoContent
				no = m[1]
			} else {
				no = m[1]
			}
			cls, _ := t.Attr("class")
			taken = cls == "replayBtn"
		}

		p = append(p, Progress{typ, name, no, taken})

		return true
	})

	if innerError != nil {
		return nil, innerError
	}

	return p, nil
}

func (c *Client) ViewNormal(progressNo string, page int, interval time.Duration) (bool, bool, error) {
	u := "https://labsafe.pknu.ac.kr/Edu/ContentsViewPop"
	data := url.Values{
		"scheduleMemberProgressNo": {progressNo},
		"currentPage":              {strconv.Itoa(page)},
	}
	resp, err := c.hc.Get(fmt.Sprintf("%s?%s", u, data.Encode()))
	if err != nil {
		return false, false, errors.Wrap(err, "http get error")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, false, errors.Wrap(err, "failed reading response body")
	}
	m := searchRegexp(`return Number\('(\d+)'\);`, string(body))
	if len(m) == 0 {
		return false, false, errors.New("failed finding matching text")
	}
	if m[1] != strconv.Itoa(page) {
		return false, false, errors.New("bad response")
	}

	time.Sleep(interval)

	u = "https://labsafe.pknu.ac.kr/Edu/ContentsViewNextProcess"
	data = url.Values{
		"scheduleMemberProgressNo": {progressNo},
		"gapTime":                  {"2000"}, // arbitary value
		"currentPage":              {strconv.Itoa(page)},
	}
	req, err := http.NewRequest("POST", u, strings.NewReader(data.Encode()))
	if err != nil {
		return false, false, errors.Wrap(err, "failed creating new http.Request")
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Referer", fmt.Sprintf("https://labsafe.pknu.ac.kr/Edu/ContentsViewPop?scheduleMemberProgressNo=%s&currentPage=%d", progressNo, page))
	resp, err = c.hc.Do(req)
	if err != nil {
		return false, false, errors.Wrap(err, "failed sending http request")
	}
	defer resp.Body.Close()

	var r struct {
		Success    bool
		IsLastPage bool
	}
	if err = json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return false, false, errors.Wrap(err, "json decode error")
	}

	return r.Success, r.IsLastPage, nil
}

func (c *Client) ViewVideo(progressNo string) (bool, error) {
	u := "https://labsafe.pknu.ac.kr/Edu/AviProcessCheck"
	data := url.Values{
		"scheduleMemberProgressNo": {progressNo},
		"currentTime":              {"30000"},
		"isEnd":                    {"true"},
	}
	resp, err := c.hc.PostForm(u, data)
	if err != nil {
		return false, errors.Wrap(err, "http post error")
	}
	defer resp.Body.Close()

	var r struct {
		IsSuccess bool
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return false, errors.Wrap(err, "json decode error")
	}

	return r.IsSuccess, nil
}

func (c *Client) ExamExploit() (bool, error) {
	no, err := c.MemberNo()
	if err != nil {
		return false, errors.Wrap(err, "failed getting member no")
	}

	u := "https://labsafe.pknu.ac.kr/Edu/ExamSend"
	data := url.Values{
		"scheduleMemberNo": {no},
	}
	for i := 0; i < 10; i++ {
		data[fmt.Sprintf("qustionAnswerList[%d].contentQuestionNo", i)] = []string{"2999"} // arbitary value
		data[fmt.Sprintf("qustionAnswerList[%d].Answer", i)] = []string{"2"}               // arbitary value
	}
	resp, err := c.hc.PostForm(u, data)
	if err != nil {
		return false, errors.Wrap(err, "http post error")
	}
	defer resp.Body.Close()

	var r struct {
		IsSuccess bool
		Point     int
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return false, errors.Wrap(err, "json decode error")
	}

	return r.IsSuccess && r.Point == 100, nil
}

func searchRegexp(pattern, str string) []string {
	re := regexp.MustCompile(pattern)
	return re.FindStringSubmatch(str)
}
