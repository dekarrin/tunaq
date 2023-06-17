package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// jsonNoContent returns an endpointResult containing an HTTP-204 along
// with a more detailed message (if desired; if none is provided it defaults to
// a generic one) that is not displayed to the user.
func jsonNoContent(internalMsg ...interface{}) endpointResult {
	internalMsgFmt := "no content"
	var msgArgs []interface{}
	if len(internalMsg) >= 1 {
		internalMsgFmt = internalMsg[0].(string)
		msgArgs = internalMsg[1:]
	}

	return jsonResponse(http.StatusNoContent, nil, internalMsgFmt, msgArgs)
}

// jsonCreated returns an endpointResult containing an HTTP-201 along
// with a more detailed message (if desired; if none is provided it defaults to
// a generic one) that is not displayed to the user.
func jsonCreated(respObj interface{}, internalMsg ...interface{}) endpointResult {
	internalMsgFmt := "created"
	var msgArgs []interface{}
	if len(internalMsg) >= 1 {
		internalMsgFmt = internalMsg[0].(string)
		msgArgs = internalMsg[1:]
	}

	return jsonResponse(http.StatusCreated, respObj, internalMsgFmt, msgArgs)
}

// jsonMethodNotAllowed returns an endpointResult containing an HTTP-400 along
// with a more detailed message (if desired; if none is provided it defaults to
// a generic one) that is not displayed to the user.
func jsonBadRequest(userMsg string, internalMsg ...interface{}) endpointResult {
	internalMsgFmt := "bad request"
	var msgArgs []interface{}
	if len(internalMsg) >= 1 {
		internalMsgFmt = internalMsg[0].(string)
		msgArgs = internalMsg[1:]
	}

	return jsonErr(http.StatusNotFound, userMsg, internalMsgFmt, msgArgs)
}

// jsonMethodNotAllowed returns an endpointResult containing an HTTP-405 along
// with a more detailed message (if desired; if none is provided it defaults to
// a generic one) that is not displayed to the user.
func jsonMethodNotAllowed(req *http.Request, internalMsg ...interface{}) endpointResult {
	internalMsgFmt := "method not allowed"
	var msgArgs []interface{}
	if len(internalMsg) >= 1 {
		internalMsgFmt = internalMsg[0].(string)
		msgArgs = internalMsg[1:]
	}

	userMsg := fmt.Sprintf("Method %s is not allowed for %s", req.Method, req.URL.Path)

	return jsonErr(http.StatusMethodNotAllowed, userMsg, internalMsgFmt, msgArgs)
}

// jsonNotFound returns an endpointResult containing an HTTP-404 response along
// with a more detailed message (if desired; if none is provided it defaults to
// a generic one) that is not displayed to the user.
func jsonNotFound(internalMsg ...interface{}) endpointResult {
	internalMsgFmt := "not found"
	var msgArgs []interface{}
	if len(internalMsg) >= 1 {
		internalMsgFmt = internalMsg[0].(string)
		msgArgs = internalMsg[1:]
	}

	return jsonErr(http.StatusNotFound, "The requested resource was not found", internalMsgFmt, msgArgs)
}

// jsonForbiddin returns an endpointResult containing an HTTP-403 response
// along with the proper WWW-Authenticate header. internalMsg is a detailed
// error message  (if desired; if none is provided it defaults to
// a generic one) that is not displayed to the user.
func jsonForbidden(internalMsg ...interface{}) endpointResult {
	internalMsgFmt := "forbidden"
	var msgArgs []interface{}
	if len(internalMsg) >= 1 {
		internalMsgFmt = internalMsg[0].(string)
		msgArgs = internalMsg[1:]
	}

	return jsonErr(http.StatusForbidden, "You don't have permission to do that", internalMsgFmt, msgArgs...)
}

// jsonUnauthorized returns an endpointResult containing an HTTP-401 response
// along with the proper WWW-Authenticate header. internalMsg is a detailed
// error message  (if desired; if none is provided it defaults to
// a generic one) that is not displayed to the user.
func jsonUnauthorized(internalMsg ...interface{}) endpointResult {
	internalMsgFmt := "unauthorized"
	var msgArgs []interface{}
	if len(internalMsg) >= 1 {
		internalMsgFmt = internalMsg[0].(string)
		msgArgs = internalMsg[1:]
	}

	return jsonErr(http.StatusUnauthorized, "You are not authorized to do that", internalMsgFmt, msgArgs...).
		withHeader("WWW-Authenticate", `Basic realm="TunaQuest server", charset="utf-8"`)
}

// jsonInternalServerError returns an endpointResult containing an HTTP-500
// response along with a more detailed message that is not displayed to the
// user. If internalMsg is provided the first argument must be a string that is
// the format string and any subsequent args are passed to Sprintf with the
// first as the format string.
func jsonInternalServerError(internalMsg ...interface{}) endpointResult {
	internalMsgFmt := "internal server error"
	var msgArgs []interface{}
	if len(internalMsg) >= 1 {
		internalMsgFmt = internalMsg[0].(string)
		msgArgs = internalMsg[1:]
	}

	return jsonErr(http.StatusInternalServerError, "An internal server error occurred", internalMsgFmt, msgArgs...)
}

// if status is http.StatusNoContent, respObj will not be read and may be nil.
// Otherwise, respObj MUST NOT be nil. If additional values are provided they
// are given to internalMsg as a format string.
func jsonResponse(status int, respObj interface{}, internalMsg string, v ...interface{}) endpointResult {
	msg := fmt.Sprintf(internalMsg, v...)
	return endpointResult{
		isJSON:      true,
		isErr:       false,
		status:      status,
		internalMsg: msg,
		resp:        respObj,
	}
}

// If additional values are provided they are given to internalMsg as a format
// string.
func jsonErr(status int, userMsg, internalMsg string, v ...interface{}) endpointResult {
	msg := fmt.Sprintf(internalMsg, v...)
	return endpointResult{
		isJSON:      true,
		isErr:       true,
		status:      status,
		internalMsg: msg,
		resp: ErrorResponse{
			Error:  userMsg,
			Status: status,
		},
	}
}

// textErr is like jsonErr but it avoids JSON encoding of any kind and writes
// the output as plain text. If additional values are provided they are given to
// internalMsg as a format string.
func textErr(status int, userMsg, internalMsg string, v ...interface{}) endpointResult {
	msg := fmt.Sprintf(internalMsg, v...)
	return endpointResult{
		isJSON:      false,
		isErr:       true,
		status:      status,
		internalMsg: msg,
		resp:        userMsg,
	}
}

type endpointResult struct {
	isErr       bool
	isJSON      bool
	status      int
	internalMsg string
	resp        interface{}
	hdrs        [][2]string
}

func (r endpointResult) withHeader(name, val string) endpointResult {
	erCopy := endpointResult{
		isErr:       r.isErr,
		isJSON:      r.isJSON,
		status:      r.status,
		internalMsg: r.internalMsg,
		resp:        r.resp,
		hdrs:        r.hdrs,
	}

	erCopy.hdrs = append(erCopy.hdrs, [2]string{name, val})
	return erCopy
}

func (r endpointResult) writeResponse(w http.ResponseWriter, req *http.Request) {
	// if this hasn't been properly created, output error directly and do not
	// try to read properties
	if r.status == 0 {
		logHttpResponse("ERROR", req, http.StatusInternalServerError, "endpoint result was never populated")
		http.Error(w, "An internal server error occurred", http.StatusInternalServerError)
		return
	}

	var respJSON []byte
	if r.isJSON && r.status != http.StatusNoContent {
		var err error
		respJSON, err = json.Marshal(r.resp)
		if err != nil {
			res := jsonErr(r.status, "An internal server error occurred", "could not marshal JSON response: "+err.Error())
			res.writeResponse(w, req)
		}
	}

	if r.isErr {
		logHttpResponse("ERROR", req, r.status, r.internalMsg)
	} else {
		logHttpResponse("INFO", req, r.status, r.internalMsg)
	}

	var respBytes []byte

	if r.isJSON {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		respBytes = respJSON
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.status != http.StatusNoContent {
			respBytes = []byte(fmt.Sprintf("%v", r.resp))
		}
	}

	for i := range r.hdrs {
		w.Header().Set(r.hdrs[i][0], r.hdrs[i][1])
	}

	w.WriteHeader(r.status)

	if r.status != http.StatusNoContent {
		w.Write(respBytes)
	}
}

func logHttpResponse(level string, req *http.Request, respStatus int, msg string) {
	if len(level) > 5 {
		level = level[0:5]
	}

	for len(level) < 5 {
		level += " "
	}

	log.Printf("%s: HTTP-%d response to %s %s: %s", level, respStatus, req.Method, req.URL.Path, msg)
}
