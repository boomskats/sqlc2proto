package parser

// QueryType represents the type of a sqlc query
type QueryType string

const (
	QueryTypeOne  QueryType = "one"
	QueryTypeMany QueryType = "many"
	QueryTypeExec QueryType = "exec"
)

// QueryMethod represents a parsed sqlc query method from the Querier interface
type QueryMethod struct {
	Name       string
	Type       QueryType
	ParamTypes []ParamType
	ReturnType string
	IsArray    bool
	Comment    string
}

// ParamType represents a parameter type
type ParamType struct {
	Name string
	Type string
}

// ServiceDefinition represents a service definition for a proto file
type ServiceDefinition struct {
	Name        string
	Description string
	Methods     []ServiceMethod
}

// ServiceMethod represents a method in a service definition
type ServiceMethod struct {
	Name            string
	Description     string
	RequestType     string
	ResponseType    string
	RequestFields   []ProtoField
	ResponseFields  []ProtoField
	OriginalQuery   *QueryMethod
	StreamingServer bool
	StreamingClient bool
}

