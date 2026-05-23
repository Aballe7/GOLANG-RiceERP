package handlers

import "fmt"

// Response is the standard return type for all handler methods.
// Every exported handler method must return this type so the JS can handle uniformly.
type Response struct {
	OK      bool        `json:"ok"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func okResponse(msg string, data interface{}) Response {
	return Response{OK: true, Message: msg, Data: data}
}

// OkResponse is the exported version for use from app.go.
func OkResponse(msg string, data interface{}) Response {
	return Response{OK: true, Message: msg, Data: data}
}

func errResponse(err error) Response {
	return Response{OK: false, Message: err.Error(), Data: nil}
}

func errMsg(msg string) Response {
	return Response{OK: false, Message: msg, Data: nil}
}

func unauthorized() Response {
	return Response{OK: false, Message: "Unauthorized. Please log in.", Data: nil}
}

func forbidden(module string) Response {
	return Response{OK: false, Message: fmt.Sprintf("Access denied: you don't have permission to access %s.", module), Data: nil}
}
