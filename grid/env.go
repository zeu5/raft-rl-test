package grid

import (
	"fmt"

	"github.com/zeu5/raft-rl-test/types"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type GridEnvironment struct {
	Height int
	Width  int
	Grids  int
	CurPos *Position
}

var _ types.Environment = &GridEnvironment{}

func NewGridEnvironment(height, width, grids int) *GridEnvironment {
	return &GridEnvironment{
		Height: height,
		Width:  width,
		Grids:  grids,
		CurPos: &Position{0, 0, 0},
	}
}

func (g *GridEnvironment) Reset() types.State {
	g.CurPos = &Position{0, 0, 0}
	return g.CurPos
}

func (g *GridEnvironment) Step(a types.Action) types.State {
	movement := a.(*Movement)
	newPos := &Position{I: g.CurPos.I, J: g.CurPos.J, K: g.CurPos.K}
	switch movement.Direction {
	case "Nothing":
	case "Up":
		newPos.I = min(g.Height-1, g.CurPos.I+1)
	case "Down":
		newPos.I = max(0, g.CurPos.I-1)
	case "Left":
		newPos.J = max(0, g.CurPos.J-1)
	case "Right":
		newPos.J = min(g.Width-1, g.CurPos.J+1)
	case "Next":
		if g.CurPos.I == g.Height-1 && g.CurPos.J == g.Width-1 {
			if g.CurPos.K < g.Grids-1 {
				newPos.I = 0
				newPos.J = 0
				newPos.K = g.CurPos.K + 1
			}
		}
	}
	g.CurPos = newPos
	return newPos
}

type Position struct {
	I int
	J int
	K int
}

var _ types.State = &Position{}

func (p *Position) Hash() string {
	return fmt.Sprintf("(%d, %d, %d)", p.I, p.J, p.K)
}

func (p *Position) Actions() []types.Action {
	if p.I == 0 && p.J == 0 {
		return []types.Action{NoMovement, NextGridMovement, MovementUp, MovementRight}
	} else if p.I == 0 {
		return []types.Action{NoMovement, NextGridMovement, MovementUp, MovementRight, MovementLeft}
	} else if p.J == 0 {
		return []types.Action{NoMovement, NextGridMovement, MovementUp, MovementRight, MovementDown}
	}
	return AllMovements
}

func DefaultStateAbstractor() types.StateAbstractor {
	return func(s types.State) string {
		return s.Hash()
	}
}

type Movement struct {
	Direction string
}

var _ types.Action = &Movement{}

func (m *Movement) Hash() string {
	return m.Direction
}

var (
	MovementUp                      = &Movement{"Up"}
	MovementDown                    = &Movement{"Down"}
	MovementLeft                    = &Movement{"Left"}
	MovementRight                   = &Movement{"Right"}
	NoMovement                      = &Movement{"Nothing"}
	NextGridMovement                = &Movement{"Next"}
	AllMovements     []types.Action = []types.Action{
		MovementUp,
		MovementDown,
		MovementLeft,
		MovementRight,
		NoMovement,
		NextGridMovement,
	}
)
