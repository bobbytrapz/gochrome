package gochrome

import (
	"context"
	"encoding/json"
	"fmt"
)

// Log function
var Log = func(string, ...interface{}) {}

// Protocol describes the DevTools Protocol
type Protocol struct {
	Domains []Domain
	Version Version
}

// Version of protocol
type Version struct {
	Major string
	Minor string
}

// VersionString gives the version string
func (p *Protocol) VersionString() string {
	return fmt.Sprintf("%s.%s", p.Version.Major, p.Version.Minor)
}

// Domain is a single protocol domain
type Domain struct {
	Domain       string
	Experimental bool
	Dependencies []string
	Types        []Type
	Commands     []Command
	Events       []Event
}

// Command from protocol
type Command struct {
	Name         string
	Description  string
	Experimental bool
	Parameters   []Parameter
	Returns      []Return
	// hack: extended
	Method string
}

// Event from protocol
type Event struct {
	Name        string
	Description string
	Parameters  []Parameter
}

// Parameter from protocol
type Parameter struct {
	Name        string
	Description string
	Optional    bool
	Type        string
	Enum        []string
	Ref         string `json:"$ref"`
	Items       Item
}

// Return from a command
type Return struct {
	Name        string
	Description string
	Optional    bool
	Type        string
	Ref         string `json:"$ref"`
	Items       Item
}

// Item describes array contents
type Item struct {
	Ref  string `json:"$ref"`
	Type string
}

// Type of variable
type Type struct {
	ID          string
	Description string
	Type        string
	Enum        []string
	Properties  []Property
}

// Property of object
type Property struct {
	Name        string
	Description string
	Optional    bool
	Type        string
	Ref         string `json:"$ref"`
	Items       Item
}

// GetProtocol from chrome
func (b *Browser) GetProtocol() (protocol Protocol) {
	res, err := b.get(context.TODO(), "/json/protocol")
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	json.NewDecoder(res.Body).Decode(&protocol)

	return
}
