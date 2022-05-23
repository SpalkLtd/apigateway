package apig_test

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	apig "github.com/SpalkLtd/apigateway"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

var testRequest string = `{
  "body": "{\"test\":\"body\"}",
  "resource": "/{proxy+}",
  "requestContext": {
    "resourceId": "123456",
    "apiId": "1234567890",
    "resourcePath": "/{proxy+}",
    "httpMethod": "POST",
    "requestId": "c6af9ac6-7b61-11e6-9a41-93e8deadbeef",
    "accountId": "123456789012",
    "identity": {
      "apiKey": null,
      "userArn": null,
      "cognitoAuthenticationType": null,
      "caller": null,
      "userAgent": "Custom User Agent String",
      "user": null,
      "cognitoIdentityPoolId": null,
      "cognitoIdentityId": null,
      "cognitoAuthenticationProvider": null,
      "sourceIp": "127.0.0.1",
      "accountId": null
    },
    "stage": "prod"
  },
  "queryStringParameters": {
    "foo": "bar"
  },
  "headers": {
    "Via": "1.1 08f323deadbeefa7af34d5feb414ce27.cloudfront.net (CloudFront)",
    "Accept-Language": "en-US,en;q=0.8",
    "CloudFront-Is-Desktop-Viewer": "true",
    "CloudFront-Is-SmartTV-Viewer": "false",
    "CloudFront-Is-Mobile-Viewer": "false",
    "X-Forwarded-For": "127.0.0.1, 127.0.0.2",
    "CloudFront-Viewer-Country": "US",
    "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
    "Upgrade-Insecure-Requests": "1",
    "X-Forwarded-Port": "443",
    "Host": "1234567890.execute-api.us-east-1.amazonaws.com",
    "X-Forwarded-Proto": "https",
    "X-Amz-Cf-Id": "cDehVQoZnx43VYQb9j2-nvCh-9z396Uhbp027Y2JvkCPNLmGJHqlaA==",
    "CloudFront-Is-Tablet-Viewer": "false",
    "Cache-Control": "max-age=0",
    "User-Agent": "Custom User Agent String",
    "CloudFront-Forwarded-Proto": "https",
    "Accept-Encoding": "gzip, deflate, sdch"
  },
  "pathParameters": {
    "proxy": "path/to/resource"
  },
  "httpMethod": "POST",
  "stageVariables": {
    "baz": "qux"
  },
  "path": "/path/to/resource"
}`

func TestRequestParsedCorrectly(t *testing.T) {
	body := []byte(testRequest)
	req := events.APIGatewayProxyRequest{}
	err := json.Unmarshal(body, &req)
	require.NoError(t, err)
	log.Println(req)
	stReq, err := apig.ToStdLibRequest(req)
	require.NoError(t, err)
	b, err := ioutil.ReadAll(stReq.Body)
	require.NoError(t, err)
	require.Equal(t, "{\"test\":\"body\"}", string(b))
	require.Equal(t, "1234567890.execute-api.us-east-1.amazonaws.com/prod", stReq.Host)
}

func TestToStdLibRequestMultiQuery(t *testing.T) {
	testData := `{
    "body": "{\"test\":\"body\"}",
    "resource": "/{proxy+}",
    "requestContext": {
      "resourceId": "123456",
      "apiId": "1234567890",
      "resourcePath": "/{proxy+}",
      "httpMethod": "POST",
      "requestId": "c6af9ac6-7b61-11e6-9a41-93e8deadbeef",
      "accountId": "123456789012",
      "identity": {
        "apiKey": null,
        "userArn": null,
        "cognitoAuthenticationType": null,
        "caller": null,
        "userAgent": "Custom User Agent String",
        "user": null,
        "cognitoIdentityPoolId": null,
        "cognitoIdentityId": null,
        "cognitoAuthenticationProvider": null,
        "sourceIp": "127.0.0.1",
        "accountId": null
      },
      "stage": "prod"
    },
    "multiValueQueryStringParameters": { "petType": [ "dog", "fish" ] },
    "headers": {
      "Via": "1.1 08f323deadbeefa7af34d5feb414ce27.cloudfront.net (CloudFront)",
      "Accept-Language": "en-US,en;q=0.8",
      "CloudFront-Is-Desktop-Viewer": "true",
      "CloudFront-Is-SmartTV-Viewer": "false",
      "CloudFront-Is-Mobile-Viewer": "false",
      "X-Forwarded-For": "127.0.0.1, 127.0.0.2",
      "CloudFront-Viewer-Country": "US",
      "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
      "Upgrade-Insecure-Requests": "1",
      "X-Forwarded-Port": "443",
      "Host": "1234567890.execute-api.us-east-1.amazonaws.com",
      "X-Forwarded-Proto": "https",
      "X-Amz-Cf-Id": "cDehVQoZnx43VYQb9j2-nvCh-9z396Uhbp027Y2JvkCPNLmGJHqlaA==",
      "CloudFront-Is-Tablet-Viewer": "false",
      "Cache-Control": "max-age=0",
      "User-Agent": "Custom User Agent String",
      "CloudFront-Forwarded-Proto": "https",
      "Accept-Encoding": "gzip, deflate, sdch"
    },
    "pathParameters": {
      "proxy": "path/to/resource"
    },
    "httpMethod": "POST",
    "stageVariables": {
      "baz": "qux"
    },
    "path": "/path/to/resource"
  }`

	body := []byte(testData)
	req := events.APIGatewayProxyRequest{}
	err := json.Unmarshal(body, &req)
	require.NoError(t, err)
	log.Println(req)
	stReq, err := apig.ToStdLibRequest(req)
	require.NoError(t, err)
	require.Equal(t, "https://1234567890.execute-api.us-east-1.amazonaws.com%2Fprod/path/to/resource?petType=dog&petType=fish", stReq.URL.String())
}

func TestToStdLibRequestV2MultiQuery(t *testing.T) {
	testData := `{
    "version": "2.0",
    "routeKey": "$default",
    "rawPath": "/my/path",
    "rawQueryString": "parameter1=value1&parameter1=value2&parameter2=value",
    "cookies": [
      "cookie1",
      "cookie2"
    ],
    "headers": {
      "header1": "value1",
      "header2": "value1,value2"
    },
    "queryStringParameters": {
      "petType": "cat,dog",
      "parameter2": "value"
    },
    "requestContext": {
      "accountId": "123456789012",
      "apiId": "api-id",
      "authentication": {
        "clientCert": {
          "clientCertPem": "CERT_CONTENT",
          "subjectDN": "www.example.com",
          "issuerDN": "Example issuer",
          "serialNumber": "a1:a1:a1:a1:a1:a1:a1:a1:a1:a1:a1:a1:a1:a1:a1:a1",
          "validity": {
            "notBefore": "May 28 12:30:02 2019 GMT",
            "notAfter": "Aug  5 09:36:04 2021 GMT"
          }
        }
      },
      "authorizer": {
        "jwt": {
          "claims": {
            "claim1": "value1",
            "claim2": "value2"
          },
          "scopes": [
            "scope1",
            "scope2"
          ]
        }
      },
      "domainName": "id.execute-api.us-east-1.amazonaws.com",
      "domainPrefix": "id",
      "http": {
        "method": "POST",
        "path": "/my/path",
        "protocol": "HTTP/1.1",
        "sourceIp": "IP",
        "userAgent": "agent"
      },
      "requestId": "id",
      "routeKey": "$default",
      "stage": "$default",
      "time": "12/Mar/2020:19:03:58 +0000",
      "timeEpoch": 1583348638390
    },
    "body": "Hello from Lambda",
    "pathParameters": {
      "parameter1": "value1"
    },
    "isBase64Encoded": false,
    "stageVariables": {
      "stageVariable1": "value1",
      "stageVariable2": "value2"
    }
  }`

	body := []byte(testData)
	req := events.APIGatewayV2HTTPRequest{}
	err := json.Unmarshal(body, &req)
	require.NoError(t, err)
	log.Println(req)
	stReq, err := apig.ToStdLibRequestV2(req)
	require.NoError(t, err)
	require.Equal(t, "petType=cat&petType=dog&parameter2=value", stReq.URL.RawQuery)
}

func TestToApigRequestMultiQuery(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://api-dev.spalk.tv/commentator-test/all?page=0&pageSize=20&reviewFilter=2&reviewFilter=5&reviewFilter=4", nil)
	require.NoError(t, err)

	apgReq, err := apig.ToApigRequest(*req)
	require.NoError(t, err)
	require.Equal(t, "", apgReq.MultiValueQueryStringParameters)
}
