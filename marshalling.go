package apig

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

//ToStdLibRequest converts the parsed json message into the format expected by the std library
func ToStdLibRequest(req events.APIGatewayProxyRequest) (*http.Request, error) {
	// spew.Fdump(os.Stderr, req)
	queryString := "?"
	for key, value := range req.QueryStringParameters {
		if len(queryString) > 1 {
			queryString += "&"
		}
		queryString += key + "=" + value
	}
	shr, err := http.NewRequest(req.HTTPMethod, "https://host"+req.Path+queryString, bytes.NewBuffer([]byte(req.Body)))
	if err != nil {
		return shr, err
	}
	shr.Host = req.Headers["Host"] + "/" + req.RequestContext.Stage
	shr.URL.Host = req.Headers["Host"] + "/" + req.RequestContext.Stage
	shr.URL.Scheme = req.Headers["CloudFront-Forwarded-Proto"]
	shr.RemoteAddr = req.RequestContext.Identity.SourceIP
	for key, values := range req.Headers {
		shr.Header.Add(key, values)
	}
	return shr, err
}

func ToApigRequest(req http.Request) (events.APIGatewayProxyRequest, error) {
	apigReq := events.APIGatewayProxyRequest{}

	parts := strings.Split(req.URL.RawPath, "/")
	apigReq.RequestContext.Stage = parts[0]
	apigReq.HTTPMethod = req.Method
	query := req.URL.Query()
	for k, v := range query {
		apigReq.QueryStringParameters[k] = v[len(v)-1]
	}
	for key, values := range req.Header {
		apigReq.Headers[key] = values[len(values)-1]
	}
	apigReq.Headers["Host"] = req.Host
	apigReq.Headers["CloudFront-Forwarded-Proto"] = req.URL.Scheme
	apigReq.RequestContext.Identity.SourceIP = req.RemoteAddr
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return apigReq, err
	}
	apigReq.Body = string(body)
	return apigReq, nil
}

func ToStdLibResponse(resp events.APIGatewayProxyResponse) http.Response {
	shr := http.Response{
		StatusCode: resp.StatusCode,
		Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(resp.Body))),
		Header:     http.Header{},
	}
	for k, v := range resp.Headers {
		shr.Header.Add(k, v)
	}
	return shr
}
