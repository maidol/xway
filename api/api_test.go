package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	// oxytest "github.com/vulcand/oxy/testutils"
	"github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/mock"
	// "github.com/stretchr/testify/suite"
)

func TestV2Status(t *testing.T) {
	router := mux.NewRouter()
	InitProxyController(nil, nil, router)
	ts := httptest.NewServer(router)
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/v2/status")
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusOK, fmt.Sprintf("resp.StatusCode = %v, want %v", resp.StatusCode, http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err, fmt.Sprintf("ioutil.ReadAll err %v", err))
	assert.Equal(t, string(body), `{"Status":"ok"}`, fmt.Sprintf("string(body) = %v, want %v", string(body), `{"Status":"ok"}`))
}

// use [go test]
// func TestV2Status(t *testing.T) {
// 	router := mux.NewRouter()
// 	InitProxyController(nil, nil, router)
// 	ts := httptest.NewServer(router)
// 	defer ts.Close()
// 	resp, err := http.Get(ts.URL + "/v2/status")
// 	if err != nil {
// 		t.Errorf("http.Get err %v", err)
// 	}
// 	defer resp.Body.Close()
// 	if resp.StatusCode != http.StatusOK {
// 		t.Errorf("resp.StatusCode = %v, want %v", resp.StatusCode, http.StatusOK)
// 	}
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		t.Errorf("ioutil.ReadAll err %v", err)
// 	}
// 	if string(body) != `{"Status":"ok"}` {
// 		t.Errorf("string(body) = %v, want %v", string(body), `{"Status":"ok"}`)
// 	}
// }

// use [. "gopkg.in/check.v1" // unittest go-check]
// func TestApi(t *testing.T) { TestingT(t) }

// type ApiSuite struct {
// 	testServer *httptest.Server
// }

// var _ = Suite(&ApiSuite{})

// func (s *ApiSuite) SetUpTest(c *C) {
// 	router := mux.NewRouter()
// 	InitProxyController(nil, nil, router)
// 	s.testServer = httptest.NewServer(router)
// }

// func (s *ApiSuite) TearDownTest(c *C) {
// 	s.testServer.Close()
// }

// func (s *ApiSuite) TestV2Status(c *C) {
// 	resp, body, err := oxytest.Get(s.testServer.URL + "/v2/status")
// 	c.Assert(err, IsNil)
// 	c.Assert(resp.StatusCode, Equals, http.StatusOK)
// 	c.Assert(string(body), Equals, `{"Status":"ok"}`)
// }
