package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"

	"github.com/rusriver/httpwrap"
	rusriverterr "github.com/rusriver/terr"
)

func main() {
	jar, err := cookiejar.New(nil)
	ifErrPanic(nil)
	http1 := httpwrap.V15HTTP{
		Retries:      2,
		RetryDelayMs: 1000,
		HTTPClient: &http.Client{
			Jar: jar,
		},
		Terr: 	&rusriverterr.Terr{},
	}

	data, err := os.ReadFile(os.Args[1])
	ifErrPanic(err)

	pReq := PostinaRequest{}
	err = json.Unmarshal(data, &pReq)
	ifErrPanic(err)

	httpReqMsg := &httpwrap.V15HTTPRequestMessage{
		Method:    pReq.Method,
		URL:       pReq.URL,
		URLParams: map[string]string{},
		Headers:   pReq.Headers,
		RawData:   pReq.Body,
	}
	for k, v := range pReq.URLParams {
		httpReqMsg.URLParams[k] = fmt.Sprintf("%v", v)
	}

	httpRespMsg, errTag := http1.ProcessRequestMessageWithRawData(httpReqMsg)
	ifErrPanic(errTag)

	pResp := &PostinaResponse{
		Code:     httpRespMsg.Code,
		CodeText: http.StatusText(httpRespMsg.Code),
		Headers:  httpRespMsg.Headers,
	}

	x := map[string]interface{}{}
	err = json.Unmarshal(httpRespMsg.RawData, &x)
	if err == nil {
		pResp.BodyJSON, err = json.MarshalIndent(x, "", "\t")
	} else {
		pResp.BodyRaw = string(httpRespMsg.RawData)
	}

	pRespBytes, err := json.MarshalIndent(pResp, "", "\t")
	ifErrPanic(err)
	fmt.Printf("%s\n", pRespBytes)

	return
}

type PostinaRequest struct {
	Method    string
	URL       string
	URLParams map[string]interface{}
	Headers   map[string]string
	Body      json.RawMessage
}
type PostinaResponse struct {
	Code     int
	CodeText string
	Headers  map[string]string
	BodyRaw  string
	BodyJSON json.RawMessage
}

func ifErrPanic(err error) {
	if err != nil {
		panic(err)
	}
}

