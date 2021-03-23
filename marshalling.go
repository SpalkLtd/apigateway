package apig

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

var re = regexp.MustCompile(`(?m)\w+.execute-api.[\w-]+.amazonaws.com`)

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

	shr.Host = req.Headers["Host"]
	shr.URL.Host = req.Headers["Host"]
	// If we are on an aws domain, add the request Stage to the host
	// (as it sneaks into the url path but is not considered in the "Path")
	if re.MatchString(req.Headers["Host"]) {
		shr.Host = shr.Host + "/" + req.RequestContext.Stage
		shr.URL.Host = shr.URL.Host + "/" + req.RequestContext.Stage
	}
	shr.URL.Scheme = req.Headers["CloudFront-Forwarded-Proto"]
	shr.RemoteAddr = req.RequestContext.Identity.SourceIP
	for key, values := range req.Headers {
		shr.Header.Add(key, values)
	}
	return shr, err
}

func ToStdLibRequestV2(req events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	queryString := "?"
	for key, value := range req.QueryStringParameters {
		if len(queryString) > 1 {
			queryString += "&"
		}
		queryString += key + "=" + value
	}
	shr, err := http.NewRequest(req.RequestContext.HTTP.Method, "https://host"+req.RawPath+queryString, bytes.NewBuffer([]byte(req.Body)))
	if err != nil {
		return shr, err
	}

	shr.Host = req.Headers["Host"]
	shr.URL.Host = req.Headers["Host"]
	// If we are on an aws domain, add the request Stage to the host
	// (as it sneaks into the url path but is not considered in the "Path")
	if re.MatchString(req.Headers["Host"]) {
		shr.Host = shr.Host + "/" + req.RequestContext.Stage
		shr.URL.Host = shr.URL.Host + "/" + req.RequestContext.Stage
	}
	shr.URL.Scheme = req.Headers["CloudFront-Forwarded-Proto"]
	shr.RemoteAddr = req.RequestContext.HTTP.SourceIP
	for key, values := range req.Headers {
		shr.Header.Add(key, values)
	}
	return shr, err
}

func ToApigRequest(req http.Request) (events.APIGatewayProxyRequest, error) {
	apigReq := events.APIGatewayProxyRequest{}

	parts := strings.Split(req.URL.Path, "/")
	if parts[1] == "dev" || parts[1] == "prod" || parts[1] == "staging" {
		apigReq.RequestContext.Stage = parts[1]
		apigReq.Path = strings.Join(parts[1:], "/")
	} else {
		apigReq.Path = req.URL.Path
	}
	apigReq.HTTPMethod = req.Method
	apigReq.Headers = make(map[string]string)
	apigReq.QueryStringParameters = make(map[string]string)
	query := req.URL.Query()
	for k, v := range query {
		apigReq.QueryStringParameters[k] = v[len(v)-1]
	}
	for key, values := range req.Header {
		apigReq.Headers[key] = strings.Join(values, ";")
	}
	apigReq.Headers["Host"] = req.Host
	apigReq.Headers["CloudFront-Forwarded-Proto"] = req.URL.Scheme
	apigReq.RequestContext.Identity.SourceIP = req.RemoteAddr
	if req.Body != nil {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return apigReq, err
		}
		apigReq.Body = string(body)
	}
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
