package raptor

type action struct {
	Name     string
	Function func(*Context) error
}

func Action(name string, function func(*Context) error) action {
	return action{
		Name:     name,
		Function: function,
	}
}
