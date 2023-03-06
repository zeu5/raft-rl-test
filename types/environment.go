package types

type Environment interface {
	Reset() State
	Step(Action) State
}

type State interface {
	Hash() string
	Actions() []Action
}

type Action interface {
	Hash() string
}
