package common

type RequestType int

const (
	GetCommands RequestType = iota
	SendResults
	ReadFileRequest
)

var typeName = map[RequestType]string{
	GetCommands:     "GetCommands",
	SendResults:     "SendResults",
	ReadFileRequest: "ReadFileRequest",
}

func (rt RequestType) String() string {
	return typeName[rt]
}

type Request struct {
	AgentID  string
	Type     RequestType
	Results  []Result
	FilePath string // For ReadFileRequest
}

type Result struct {
	CommandID  string
	ReturnCode int
	Output     []byte
}
