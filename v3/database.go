package raptor

type DatabaseConnector interface {
	Init() error
	Conn() any
}
