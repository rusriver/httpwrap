package httpwrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	rusriverterr "github.com/rusriver/terr"
)

type V15HTTP struct {
	Retries                 int
	RetryDelayMs            int
	HTTPHeaders             map[string]string
	LastResponseHTTPHeaders map[string]string
	HTTPClient              *http.Client
	OkHTTPResponseCodes     []int
	Terr                    *rusriverterr.Terr
}

//type Base struct {
//	Retries                 int
//	RetryDelayMs            int
//	HTTPHeaders             map[string]string
//	LastResponseHTTPHeaders map[string]string
//	HTTPClient              *http.Client
//	OkHTTPResponseCodes     []int
//	Terr                    *rusriverterr.Terr
//}

//func NewBase() (base *Base) {
//	return
//}

type V15HTTPRequestMessage struct {
	Method    string
	URL       string
	URLParams map[string]string
	Headers   map[string]string
	RawData   []byte
	// chDown IS NOT part of this; please wrap this into higher level struct, if you need
	// to pass chDown et al.
}
type V15HTTPResponseMessage struct {
	Code    int
	Headers map[string]string
	RawData []byte
}

func (v *V15HTTP) ProcessRequestMessageWithRawData(
	reqMsg *V15HTTPRequestMessage,
) (
	respMsg *V15HTTPResponseMessage,
	tagErr rusriverterr.TagErrorer,
) {
	v.HTTPHeaders = reqMsg.Headers

	code, respBody, tagErr2 := v.RequestResponseRawData(
		reqMsg.Method,
		reqMsg.URL,
		reqMsg.URLParams,
		reqMsg.RawData,
	)
	tagErr = tagErr2
	if tagErr != nil {
		tagErr.AddTrace()
		return
	}
	respMsg = &V15HTTPResponseMessage{
		Code:    code,
		Headers: v.LastResponseHTTPHeaders,
		RawData: respBody,
	}
	return
}

func (v *V15HTTP) RequestResponseJSON(
	method string,
	url string,
	urlParams map[string]string,
	reqBody interface{},
	respBodyRef interface{},
) (
	code int,
	tagErr rusriverterr.TagErrorer,
) {
	myTags := []string{"V15HTTP", "RequestResponseJSON()"}
	for retry := 0; ; retry++ {
		if retry > 0 {
			if tagErr != nil {
				fmt.Println(tagErr.Error())
			}
			fmt.Printf(" V15HTTP RequestResponseJSON() failed; retry %v of %v; sleep %v ms... ", retry, v.Retries, v.RetryDelayMs)
			time.Sleep(time.Duration(v.RetryDelayMs) * time.Millisecond)
		}
		if retry >= v.Retries {
			return
		}
		jsonReq, err := json.Marshal(reqBody)
		if err != nil {
			tagErr = v.Terr.NewTaggedErrorFrom(append([]string{"JSON", "Request"}, myTags...), err)
			return
		}

		var respData []byte
		code, respData, tagErr = v.RequestResponseRawData(method, url, urlParams, jsonReq)
		if tagErr != nil {
			tagErr.AddTrace()
			// do not retry this, it's already retried
			return
		}

		if respBodyRef != nil {
			err = json.Unmarshal(respData, respBodyRef)
			if err != nil {
				tagErr = v.Terr.NewTaggedErrorFrom(append([]string{"JSON", "Response"}, myTags...), err)
				return
			}
		}

		if len(v.OkHTTPResponseCodes) == 0 {
			// if not set, imply any is OK
			return
		}
		for _, okCode := range v.OkHTTPResponseCodes {
			if code == okCode {
				return
			}
		}
		tagErr = v.Terr.NewError(append([]string{"HTTPCode", "Response"}, myTags...), "Response Status Code (%v) isn't in the OK set", code)
	} // retry loop
}

func (v *V15HTTP) RequestResponseRawData(
	method string,
	url string,
	urlParams map[string]string,
	reqBody []byte,
) (
	code int,
	respBody []byte,
	tagErr rusriverterr.TagErrorer,
) {
	myTags := []string{"V15HTTP", "RequestResponseRawData()"}
	for retry := 0; ; retry++ {
		if retry > 0 {
			if tagErr != nil {
				fmt.Println(tagErr.Error())
			}
			fmt.Printf(" V15HTTP RequestResponseRawData() failed; retry %v of %v; sleep %v ms... ", retry, v.Retries, v.RetryDelayMs)
			time.Sleep(time.Duration(v.RetryDelayMs) * time.Millisecond)
		}
		if retry >= v.Retries {
			return
		}

		bodyIoReader := bytes.NewBuffer(reqBody)

		req, err := http.NewRequest(method, url, bodyIoReader)
		if err != nil {
			tagErr = v.Terr.NewTaggedErrorFrom(append([]string{"HTTP", "Request"}, myTags...), err)
			return
		}

		for k, v := range v.HTTPHeaders {
			req.Header.Set(k, v)
		}

		if urlParams != nil {
			q := req.URL.Query()
			for k, v := range urlParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()
		}

		resp, err := v.HTTPClient.Do(req)
		if err != nil {
			tagErr = v.Terr.NewTaggedErrorFrom(append([]string{"HTTP", "Request"}, myTags...), err)
			continue
		}

		// get response headers
		v.LastResponseHTTPHeaders = make(map[string]string, 20)
		for hName, hVal := range resp.Header {
			if len(hVal) > 0 {
				v.LastResponseHTTPHeaders[hName] = hVal[0]
			}
		}

		respBody, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			tagErr = v.Terr.NewTaggedErrorFrom(append([]string{"ioutil", "Response"}, myTags...), err)
			continue
		}
		_ = resp.Body.Close()

		code = resp.StatusCode
		break
	}
	return
}
