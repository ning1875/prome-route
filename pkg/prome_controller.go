package pkg

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"regexp"
	"io/ioutil"

	"fmt"
	log "github.com/sirupsen/logrus"
	"encoding/json"
	"github.com/spf13/viper"
	"errors"
)

func Routes(r *gin.Engine) {

	qApi := r.Group("/")
	qApi.GET("/api/v1/query_range", promeRangequery)
	qApi.GET("/api/v1/query", promeInstancequery)
	qApi.GET("/api/v1/series", promeSeriesQuery)

}

type RangeQuery struct {
	Query string `form:"query" binding:"required"`
	Start string `form:"start" binding:"required"`
	End   string `form:"end"   binding:"required"`
	Step  string `form:"step"  binding:"required"`
}

type InstanceQuery struct {
	Query string `form:"query" binding:"required"`
	Time  string `form:"time" binding:"required"`
}

type SeriesQuery struct {
	//Match []string `form:"match[]" binding:"required"`
	Match []string `form:"match[]" binding:"required"`
	Start string `form:"start" binding:"required"`
	End   string `form:"end"   binding:"required"`
}

func promeSeriesQuery(c *gin.Context) {
	d := SeriesQuery{}

	matchers, ok := c.Request.URL.Query()["match[]"]
	if !ok{
		c.String(http.StatusBadRequest, "wrong args match[]")
		return
	}



	err, keyName, labelName, targetProme, newQuery := regexCommon(matchers[0])
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	resp := seriesQueryGetData(targetProme, newQuery, d.Start, d.End)
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err.Error())
		c.String(http.StatusInternalServerError, fmt.Sprintf(`target prome %s error by %s=%s `, targetProme, keyName, labelName))
		return
	}
	var respInterface interface{}
	_ = json.Unmarshal(respBytes, &respInterface)

	c.JSON(resp.StatusCode, respInterface)
	return

}

func promeInstancequery(c *gin.Context) {

	d := InstanceQuery{}
	err := c.Bind(&d)
	if err != nil {
		c.String(http.StatusBadRequest, "wrong args")
		return
	}
	if d.Query == "1+1" {
		c.String(http.StatusOK, "yes")
		return
	}

	err, keyName, labelName, targetProme, newQuery := regexCommon(d.Query)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	resp := instanceQueryGetData(targetProme, newQuery, d.Time)
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err.Error())
		c.String(http.StatusInternalServerError, fmt.Sprintf(`target prome %s error by %s=%s `, targetProme, keyName, labelName))
		return
	}
	var respInterface interface{}
	_ = json.Unmarshal(respBytes, &respInterface)

	c.JSON(resp.StatusCode, respInterface)
	return
}

func regexCommon(query string) (err error, keyName, labelName, targetProme, newQuery string) {
	var re *regexp.Regexp
	fmt.Println(query)


	var reReplace = regexp.MustCompile(QueryPlaceRegStr)
	var regA = regexp.MustCompile(RouteLabelRegStr)
	var regB = regexp.MustCompile(RouteLabelRegStrP)

	if regA.MatchString(query){
		re = regA
	}else if regB.MatchString(query){
		re = regB
	}else {
		err = errors.New(fmt.Sprintf(`正则匹配失败 %s or %s `, RouteLabelRegStr, RouteLabelRegStrP))
		return
	}

	match := re.FindStringSubmatch(query)


	result := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" { // 第一个分组为空（也就是整个匹配）
			result[name] = match[i]
		}
	}

	keyName = viper.GetString("replace_label_name")
	labelName, loaded := result[keyName]

	if !loaded {
		err = errors.New(fmt.Sprintf(`必须含有tag %s,eg:%s="value" `, keyName, keyName))
		return
	}
	targetProme, loaded = RromeServerMap[labelName]
	if !loaded {
		err = errors.New(fmt.Sprintf(`target prome not found in config map by %s=%s `, keyName, labelName))
		return
	}

	newQuery = re.ReplaceAllString(query, "")
	newQuery = reReplace.ReplaceAllString(newQuery, "{")
	log.Infof("[route_res][%s=%s][remote_addr:%s][old_expr:%s][new_expr:%s]",
		keyName,
		labelName,
		targetProme,

		query,
		newQuery,

	)
	return
}

func promeRangequery(c *gin.Context) {

	d := RangeQuery{}
	err := c.Bind(&d)
	if err != nil {
		c.String(http.StatusBadRequest, "wrong args")
		return
	}

	err, keyName, labelName, targetProme, newQuery := regexCommon(d.Query)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	resp := rangeQueryGetData(targetProme, newQuery, d.Start, d.End, d.Step)
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err.Error())
		c.String(http.StatusInternalServerError, fmt.Sprintf(`target prome %s error by %s=%s `, targetProme, keyName, labelName))
		return
	}
	var respInterface interface{}
	_ = json.Unmarshal(respBytes, &respInterface)

	c.JSON(resp.StatusCode, respInterface)
	return
}

func seriesQueryGetData(addr, query, start, end string) *http.Response {
	newAddr := fmt.Sprintf("http://%s/api/v1/series", addr)
	req, err := http.NewRequest("GET", newAddr, nil)

	q := req.URL.Query()
	q.Add("match[]", query)
	q.Add("start", start)
	q.Add("end", end)

	req.URL.RawQuery = q.Encode()
	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("rangeQueryGetData_error:url:%s,err:%+v", addr, err)
		return nil
	}
	return resp

}

func rangeQueryGetData(addr, query, start, end, step string) *http.Response {
	newAddr := fmt.Sprintf("http://%s/api/v1/query_range", addr)
	req, err := http.NewRequest("GET", newAddr, nil)

	q := req.URL.Query()
	q.Add("query", query)
	q.Add("start", start)
	q.Add("end", end)
	q.Add("step", step)

	req.URL.RawQuery = q.Encode()
	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("rangeQueryGetData_error:url:%s,err:%+v", addr, err)
		return nil
	}
	return resp

}

func instanceQueryGetData(addr, query, time string) *http.Response {
	newAddr := fmt.Sprintf("http://%s/api/v1/query", addr)
	req, err := http.NewRequest("GET", newAddr, nil)

	q := req.URL.Query()
	q.Add("query", query)
	q.Add("time", time)

	req.URL.RawQuery = q.Encode()
	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("rangeQueryGetData_error:url:%s,err:%+v", addr, err)
		return nil
	}
	return resp

}
