package apig

//This library is designed to help marshal requests inside a lambda using the apex shim.
//Formats are documented at http://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-set-up-simple-proxy.html#api-gateway-simple-proxy-for-lambda-input-format

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime/debug"
	"strings"

	apex "github.com/apex/go-apex"
)

//Context contains the information from the http request that is being forwarded by apigateway
type Context struct {
	ApiId        string `json:"apiId"`
	Stage        string
	SourceIp     string `json:"sourceIp"`
	Identity     Identity
	RequestId    string `json:"requestId"`
	ResourceId   string `json:"resourceId"`
	ResourcePath string `json:"resourcePath"`
}

//Identity contains information about the user who made the request
type Identity struct {
	AccountId    string `json:"accountId"`
	ApiKey       string `json:"apiKey"`
	Caller       string
	AuthProvider string `json:"cognitoAuthenticationProvider"`
	AuthType     string `json:"cognitoAuthenticationType"`
	IdentId      string `json:"cognitoIdentityId"`
	IdentPoolId  string `json:"cognitoIdentityPoolId"`
	User         string
	UserAgent    string `json:"userAgent"`
	UserArn      string `json:"userArn"`
}

//Request is the format of the apigateway request passed to the lambda
type Request struct {
	Resource              string
	Path                  string
	HttpMethod            string
	Headers               map[string]string
	QueryStringParameters map[string]string
	PathParameters        map[string]string
	StageVariables        map[string]string
	Context               Context `json:"requestContext"`
	Body                  string
	IsBase64              bool
}

// Response format expected by apigateway from a lambda proxy integration
type Response struct {
	StatusCode int               `json:"statusCode"`
	Body       string            `json:"body,omitempty"`
	RequestId  string            `json:"-"` //`json:"requestId"`
	Error      string            `json:"-,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

//Respond will produce a response that will get formatted such that apigateway will modify it's response to the browser
func Respond(body interface{}, status int, req Request, err error) (Response, error) {
	// log.Println("Entering Respond")
	if reflect.TypeOf(body).Kind() == reflect.Func {
		log.Println("Unsuported return type")
	}
	debug.PrintStack()
	bodyBytes, jsonerr := json.Marshal(body)
	if jsonerr != nil {
		log.Println(jsonerr.Error())
	}
	resp := Response{
		RequestId:  req.Context.RequestId,
		StatusCode: status,
		Body:       fmt.Sprintf("%s", bodyBytes),
	}
	if err != nil {
		if status == 200 {
			resp.StatusCode = 500
		}
		if body == nil {
			resp.Body = err.Error()
		}
		log.Println(err.Error())
	}
	if status == 0 {
		resp.StatusCode = 200
	}
	if resp.Body == "null" {
		resp.Body = ""
	}
	resp.Headers = map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "DELETE,GET,HEAD,OPTIONS,PATCH,POST,PUT",
		"Access-Control-Allow-Headers": "Content-Type,Authorization,X-Amz-Date,X-Api-Key,X-Amz-Security-Token",
		"Content-Type":                 "application/json",
	}
	return resp, nil
}

//RespondHTTP will marshall the response body as JSON and write it to the response writer
//This function signature was chosen to make it substitutable for http.Error
//This does not end the requset, but does write the header. Care should be taken to close the response after this has been called
func RespondHTTP(rw http.ResponseWriter, body interface{}, status int) {
	// log.Println("Entering RespondHTTP")
	if body != nil {
		if reflect.TypeOf(body).Kind() == reflect.Func {
			log.Println("Unsuported return type")
			debug.PrintStack()
			return
		}
		if err, ok := body.(error); ok {
			if status < 400 {
				status = 500
			}
			log.Printf("Writing %v", err.Error())
			http.Error(rw, err.Error(), status)
			return
		}
		var bodyBytes []byte
		if bodyString, ok := body.(string); ok {
			bodyBytes = []byte(bodyString)
		} else {
			jsonBytes, jsonerr := json.Marshal(body)
			if jsonerr != nil {
				log.Println(jsonerr.Error())
				http.Error(rw, "Error marshalling reponse", http.StatusInternalServerError)
			}
			bodyBytes = jsonBytes
		}
		rw.WriteHeader(status)
		written, err := rw.Write(bodyBytes)
		if err != nil {
			log.Println(err.Error())
		}
		if written != len(bodyBytes) {
			log.Println("Unable to finish writing body: " + string(bodyBytes))
		}
	} else {
		rw.Header().Set("Content-Type", "plaintext")
		rw.WriteHeader(status)
	}
	// log.Println("Exiting RespondHTTP")
}

//ToStdLibRequest converts the parsed json message into the format expected by the std library
func (req Request) ToStdLibRequest() (*http.Request, error) {
	// spew.Fdump(os.Stderr, req)
	queryString := "?"
	for key, value := range req.QueryStringParameters {
		if len(queryString) > 1 {
			queryString += "&"
		}
		queryString += key + "=" + value

	}
	shr, err := http.NewRequest(req.HttpMethod, "https://host"+req.Path+queryString, bytes.NewBuffer([]byte(req.Body)))
	if err != nil {
		return shr, err
	}
	shr.Host = req.Headers["Host"] + "/" + req.Context.Stage
	shr.URL.Host = req.Headers["Host"] + "/" + req.Context.Stage
	shr.URL.Scheme = req.Headers["CloudFront-Forwarded-Proto"]
	shr.RemoteAddr = req.Context.SourceIp
	for key, values := range req.Headers {
		shr.Header.Add(key, values)
	}
	return shr, err
}

//ResponseWriter implements the net/http ResponseWriter interface for using stdlib compliant server libraries with apigateway and lambdas
type ResponseWriter struct {
	resp   Response
	body   bytes.Buffer
	header http.Header
}

//Header returns the map that will be sent with WriteHeader
func (rw *ResponseWriter) Header() http.Header {
	if rw.header == nil {
		rw.header = make(map[string][]string)
	}
	return rw.header
}

func (rw *ResponseWriter) Write(data []byte) (int, error) {
	// log.Println("Called ResponseWriter.Write()")
	return rw.body.Write(data)
}

//WriteHeader sets the response code in the embeded response object
//To be compliant with the http spec for the interface it should write the headers to the client, but we can't control that
func (rw *ResponseWriter) WriteHeader(status int) {
	rw.resp.StatusCode = status
	//can't actually write the headers out before we return :(
}

//GetResponse formats the net/http response to how the response is expected by apigateway
func (rw *ResponseWriter) GetResponse() (Response, error) {
	// log.Println("Entering GetResponse()")
	rw.resp.Body = rw.body.String()
	rw.resp.Headers = make(map[string]string, len(rw.header))
	for key, values := range rw.header {
		rw.resp.Headers[key] = strings.Join(values, ",")
	}
	// log.Printf("Exiting GetResponse and returning %v", rw.resp)
	return rw.resp, nil
}

//Serve handles and responds to the requests using a net/http handler
func Serve(req Request, handler http.Handler) (Response, error) {
	// log.Println("Entering Serve")
	// defer func() { log.Println("Exiting Serve") }()
	shr, err := req.ToStdLibRequest()
	if err != nil {
		log.Println(err.Error())
		return Respond(nil, 500, req, err)
	}
	rw := ResponseWriter{}
	handler.ServeHTTP(&rw, shr)
	return rw.GetResponse()
}

//StartApex starts the apex server than marshals requests in/out of the apex shim using stdin/stdout
func StartApex(handler http.Handler) {
	// log.Println("Starting Apex server")
	// defer func() { log.Println("Apex server stopped") }()
	apex.HandleFunc(func(event json.RawMessage, ctx *apex.Context) (interface{}, error) {
		var req Request
		// log.Println("Entering Apex event handler")
		// defer func() { log.Println("Exiting Apex event handler") }()
		if err := json.Unmarshal(event, &req); err != nil {
			log.Println(err.Error())
			log.Println(string(event))
			return Respond(nil, 401, req, err)
		}
		for k, v := range req.StageVariables {
			os.Setenv(k, v)
		}
		resp, err := Serve(req, handler)
		if err != nil {
			log.Println(err.Error())
		}
		return resp, err

	})
}
