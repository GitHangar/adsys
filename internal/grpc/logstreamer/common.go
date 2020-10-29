package log

//go:generate protoc --go_out=. --go_opt=paths=source_relative log.proto

const (
	logIdentifier = "LOGSTREAMER_MSG"

	clientIDKey         = "ClientID"
	clientWantCallerKey = "ClientWantCallery"
)