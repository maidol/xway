package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"strings"

	en "xway/engine"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
)

type ProxyController struct {
	ng    en.Engine
	stats en.StatsProvider
}

func InitProxyController(ng en.Engine, stats en.StatsProvider, router *mux.Router) {
	c := &ProxyController{ng: ng, stats: stats}

	router.NotFoundHandler = http.HandlerFunc(c.handleError)

	router.HandleFunc("/v2/status", handlerWithBody(c.getStatus)).Methods("GET")
	router.HandleFunc("/v2/gc", handlerWithBody(c.gc)).Methods("GET")
	router.HandleFunc("/v2/stats", handlerWithBody(c.getStats)).Methods("GET")
	router.HandleFunc("/v2/router/restore", handlerWithBody(c.restoreRouter)).Methods("GET")
	router.HandleFunc("/v2/db/reset", handlerWithBody(c.resetDB)).Methods("GET")
	router.HandleFunc("/v2/apps/restore", handlerWithBody(c.restoreApps)).Methods("GET")
	router.HandleFunc("/v2/getfrontends", handlerWithBody(c.getFrontends)).Methods("GET")
}

func (pc *ProxyController) handleError(w http.ResponseWriter, r *http.Request) {
	sendResponse(w, Response{"message": "Object not found"}, http.StatusNotFound)
}

func (pc *ProxyController) getStatus(w http.ResponseWriter, r *http.Request, params map[string]string, body []byte) (interface{}, error) {
	return Response{
		"Status": "ok",
	}, nil
}

func (pc *ProxyController) gc(w http.ResponseWriter, r *http.Request, params map[string]string, body []byte) (interface{}, error) {
	runtime.GC()
	return Response{
		"Status": "ok",
	}, nil
}

func (pc *ProxyController) getStats(w http.ResponseWriter, r *http.Request, params map[string]string, body []byte) (interface{}, error) {
	registry := pc.ng.GetRegistry()
	db := registry.GetDBPool().Stats()
	rds := registry.GetRedisPool().Stats()
	tr := registry.GetTransport()
	proxy := map[string]interface{}{"maxIdleConns": tr.MaxIdleConns, "maxIdleConncPerHost": tr.MaxIdleConnsPerHost, "idleTimeout": tr.IdleConnTimeout}
	gcount := runtime.NumGoroutine()
	stats := map[string]interface{}{"gcount": gcount, "serviceOptions": registry.GetSvcOptions(), "db": db, "redis": rds, "proxy": proxy}
	return stats, nil
}

func (pc *ProxyController) getFrontends(w http.ResponseWriter, r *http.Request, params map[string]string, body []byte) (interface{}, error) {
	router := pc.ng.GetRegistry().GetRouter()
	return router.GetFrontends(), nil
}

type routeData struct {
	// routeId
}

func (pc *ProxyController) restoreRouter(w http.ResponseWriter, r *http.Request, params map[string]string, body []byte) (interface{}, error) {
	// 读取路由表(from mysql)
	db := pc.ng.GetRegistry().GetDBPool()
	rows, err := db.Query("select routeId, domainHost, routeUrl, redirectHost, forwardUrl, type, config, status from apiroutes")
	if err != nil {
		return nil, err
	}
	arr, err := scanMapString(rows)
	if err != nil {
		return nil, err
	}
	// 加载到engine
	err = pc.ng.ReloadFrontendsFromDB(arr)
	if err != nil {
		return nil, fmt.Errorf("pc.ng.ReloadFrontendsFromDB failure: %v", err)
	}
	return arr, nil
}

func (pc *ProxyController) resetDB(w http.ResponseWriter, r *http.Request, params map[string]string, body []byte) (interface{}, error) {
	registry := pc.ng.GetRegistry()
	u := registry.GetSvc()
	err := u.ResetDB()
	if err != nil {
		return nil, err
	}
	return Response{
		"Status": "ok",
	}, nil
}

func (pc *ProxyController) restoreApps(w http.ResponseWriter, r *http.Request, params map[string]string, body []byte) (interface{}, error) {
	registry := pc.ng.GetRegistry()
	// 读取
	db := registry.GetDBPool()
	rows, err := db.Query("select clientId, clientName, groupId, publicKey, privateKey, domainUrl, redirectUrl, config, status, createDate, updateDate from apps")
	if err != nil {
		return nil, err
	}
	result, err := scanMapString(rows)
	if err != nil {
		return nil, err
	}
	// 导入
	rdc := registry.GetRedisPool().Get()
	defer rdc.Close()
	for _, val := range result {
		args := redis.Args{}.Add("cw:app:" + val["clientId"]).AddFlat(val)
		log.Printf("[restoreApp] HMSET %v\n", args)
		err := rdc.Send("HMSET", args...)
		if err != nil {
			log.Printf("[restoreApp] Send HMSET err: %v\n", err)
		}
	}
	err = rdc.Flush()
	if err != nil {
		log.Printf("[restoreApp] rdc.Flush err: %v\n", err)
		return result, err
	}
	return result, nil
}

func scanMapString(rows *sql.Rows) ([]map[string]string, error) {
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	arr := []map[string]string{}
	for rows.Next() {
		row := map[string]string{}
		err := rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}

		var value string
		for i, col := range values {
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			fmt.Println(columns[i], ": ", value)
			row[columns[i]] = value
		}
		fmt.Println("-------------------------------")
		arr = append(arr, row)
	}
	if err := rows.Err(); err != nil {
		// error that was encountered during iteration
		return nil, err
	}
	return arr, nil
}

type Response map[string]interface{}

type handlerWithBodyFn func(http.ResponseWriter, *http.Request, map[string]string, []byte) (interface{}, error)

func handlerWithBody(fn handlerWithBodyFn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := parseForm(r); err != nil {
			sendResponse(w, fmt.Sprintf("failed to parse request from, err=%v", err), http.StatusInternalServerError)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			sendResponse(w, fmt.Sprintf("failed to read request body, err=%v", err), http.StatusInternalServerError)
			return
		}

		rs, err := fn(w, r, mux.Vars(r), body)
		if err != nil {
			var status int
			switch err.(type) {
			default:
				status = http.StatusInternalServerError

			}
			sendResponse(w, Response{"message": err.Error()}, status)
			return
		}
		sendResponse(w, rs, http.StatusOK)
	}
}

func parseForm(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") == true {
		return r.ParseMultipartForm(0)
	}
	return r.ParseForm()
}

func sendResponse(w http.ResponseWriter, response interface{}, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	marshalledResponse, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Failed to marshal response: %v %v", response, err)))
		return
	}
	w.WriteHeader(status)
	w.Write(marshalledResponse)
}
