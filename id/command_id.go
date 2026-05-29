package id

type commandMarker struct{}

type CommandID = Of[commandMarker]

func NewCommandID() CommandID {
	return New[commandMarker]()
}

func ParseCommandID(s string) (CommandID, error) {
	return Parse[commandMarker](s)
}

func MustParseCommandID(s string) CommandID {
	return MustParse[commandMarker](s)
}
