package apig

//This library is designed to help marshal requests inside a lambda using the apex shim.
//Formats are documented at http://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-set-up-simple-proxy.html#api-gateway-simple-proxy-for-lambda-input-format

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"unicode"

	apex "github.com/apex/go-apex"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var ErrNoHandler = errors.New("No handler defined for event of that type")

//Respond will produce a response that will get formatted such that apigateway will modify it's response to the browser
func Respond(body interface{}, status int, req events.APIGatewayProxyRequest, err error) (events.APIGatewayProxyResponse, error) {
	// log.Println("Entering Respond")
	if body != nil && reflect.TypeOf(body).Kind() == reflect.Func {
		log.Println("Unsuported return type")
	}
	debug.PrintStack()
	bodyBytes, jsonerr := json.Marshal(body)
	if jsonerr != nil {
		log.Println(jsonerr.Error())
	}
	resp := events.APIGatewayProxyResponse{
		// RequestID:  req.Context.RequestId,
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

//RespondvV2 will produce a response that will get formatted such that apigateway will modify it's response to the browser
func RespondvV2(body interface{}, status int, req events.APIGatewayV2HTTPRequest, err error) (events.APIGatewayV2HTTPResponse, error) {
	if body != nil && reflect.TypeOf(body).Kind() == reflect.Func {
		log.Println("Unsuported return type")
	}
	bodyBytes, jsonerr := json.Marshal(body)
	if jsonerr != nil {
		log.Println(jsonerr.Error())
	}
	resp := events.APIGatewayV2HTTPResponse{
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

//ResponseWriterV2 implements the net/http ResponseWriterV2 interface for using stdlib compliant server libraries with apigatewayv2 and lambdas
type ResponseWriterV2 struct {
	resp   events.APIGatewayV2HTTPResponse
	body   bytes.Buffer
	header http.Header
}

//Header returns the map that will be sent with WriteHeader
func (rw *ResponseWriterV2) Header() http.Header {
	if rw.header == nil {
		rw.header = make(map[string][]string)
	}
	return rw.header
}

func (rw *ResponseWriterV2) Write(data []byte) (int, error) {
	return rw.body.Write(data)
}

//WriteHeader sets the response code in the embeded response object
//To be compliant with the http spec for the interface it should write the headers to the client, but we can't control that
func (rw *ResponseWriterV2) WriteHeader(status int) {
	rw.resp.StatusCode = status
	//can't actually write the headers out before we return :(
}

//GetResponse formats the net/http response to how the response is expected by apigateway
func (rw *ResponseWriterV2) GetResponse() (events.APIGatewayV2HTTPResponse, error) {
	rw.resp.Body = rw.body.String()
	rw.resp.Headers = make(map[string]string, len(rw.header))
	for key, values := range rw.header {
		if strings.ToLower(key) == "set-cookie" {
			for i, v := range values {
				rw.resp.Headers[setCookieCasing(i)] = v
			}
		} else {
			rw.resp.Headers[key] = strings.Join(values, ",")
		}
	}
	return rw.resp, nil
}

//ResponseWriter implements the net/http ResponseWriter interface for using stdlib compliant server libraries with apigateway and lambdas
type ResponseWriter struct {
	resp   events.APIGatewayProxyResponse
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
func (rw *ResponseWriter) GetResponse() (events.APIGatewayProxyResponse, error) {
	rw.resp.Body = rw.body.String()
	rw.resp.Headers = make(map[string]string, len(rw.header))
	for key, values := range rw.header {
		if strings.ToLower(key) == "set-cookie" {
			for i, v := range values {
				rw.resp.Headers[setCookieCasing(i)] = v
			}
		} else {
			rw.resp.Headers[key] = strings.Join(values, ",")
		}
	}
	return rw.resp, nil
}

const SET_COOKIE = "setcookie"

func setCookieCasing(i int) string {
	returnVal := ""
	if i == 0 {
		return "set-cookie"
	}
	j := int64(512 - i%512)
	strRep := strconv.FormatInt(j, 2)
	for k := 0; k < len(strRep); k++ {
		if strRep[k] == []byte("1")[0] {
			returnVal = returnVal + toggleCase(SET_COOKIE[k])
		} else {
			returnVal = returnVal + string(SET_COOKIE[k])
		}
		if k == 2 {
			returnVal = returnVal + "-"
		}
	}
	return returnVal
}

func toggleCase(a byte) string {
	if unicode.IsUpper(rune(a)) {
		return strings.ToLower(string(a))
	}
	return strings.ToUpper(string(a))

}

//ServeV2 handles and responds to the requests using a net/http handler
func ServeV2(req events.APIGatewayV2HTTPRequest, handler http.Handler) (events.APIGatewayV2HTTPResponse, error) {
	shr, err := ToStdLibRequestV2(req)
	if err != nil {
		log.Println(err.Error())
		return RespondvV2(nil, 500, req, err)
	}
	rw := ResponseWriterV2{}
	handler.ServeHTTP(&rw, shr)
	return rw.GetResponse()
}

//Serve handles and responds to the requests using a net/http handler
func Serve(req events.APIGatewayProxyRequest, handler http.Handler) (events.APIGatewayProxyResponse, error) {
	// log.Println("Entering Serve")
	// defer func() { log.Println("Exiting Serve") }()
	shr, err := ToStdLibRequest(req)
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
	apex.HandleFunc(func(event json.RawMessage, ctx *apex.Context) (interface{}, error) {
		var req events.APIGatewayProxyRequest
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

//StartLambda ...
func StartLambda(handler http.Handler, fallback lambdaHandlerFunc) {
	lambda.Start(LambdaHandler(handler, fallback))
}

type lambdaHandlerFunc func(event json.RawMessage) (interface{}, error)

//LambdaHandler ...
func LambdaHandler(handler http.Handler, fallback lambdaHandlerFunc) lambdaHandlerFunc {
	return func(event json.RawMessage) (interface{}, error) {
		var err error
		var apigEvent events.APIGatewayProxyRequest
		if err = json.Unmarshal(event, &apigEvent); err == nil && apigEvent.Path != "" {
			for k, v := range apigEvent.StageVariables {
				os.Setenv(k, v)
			}
			resp, err := Serve(apigEvent, handler)
			if err != nil {
				log.Println(err.Error())
			}
			return resp, err
		}
		if fallback != nil {
			return fallback(event)
		}
		return nil, ErrNoHandler
	}
}
