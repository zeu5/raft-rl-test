package explorer

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Runs the main interactive loop
func (e *Explorer) Interact() {
	fmt.Printf("%s", e.header())
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s", e.prompt())

		optionS, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Invalid input! Try again")
			continue
		}
		option, err := strconv.Atoi(strings.Replace(optionS, "\n", "", -1))
		if err != nil {
			fmt.Println("Invalid input! Try again")
			continue
		}
		fmt.Println("------------------------------------")
		switch option {
		case 1:
			fmt.Printf("%s", e.getInitialStates())
		case 2:
			fmt.Printf("Enter the state key: ")
			stateK, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Invalid input! Try again")
				continue
			}
			fmt.Printf("%s", e.getQValues(strings.Replace(stateK, "\n", "", -1)))
		case 3:
			fmt.Printf("Enter the state key: ")
			stateK, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Invalid input! Try again")
				continue
			}
			fmt.Printf("%s", e.getFullState(strings.Replace(stateK, "\n", "", -1)))
		case 4:
			fmt.Printf("Enter trace number (1-%d): ", len(e.Traces))
			traceNoS, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Invalid input! Try again")
				continue
			}
			traceNo, err := strconv.Atoi(strings.Replace(traceNoS, "\n", "", -1))
			if err != nil {
				fmt.Println("Invalid input! Not a number. Try again")
				continue
			}
			if traceNo < 1 || traceNo > len(e.Traces) {
				fmt.Printf("Invalid input! Should be between (1-%d). Try again\n", len(e.Traces))
				continue
			}
			e.interactTrace(traceNo-1, reader)
		case 5:
			fmt.Println("Quitting! Thank you")
			return
		default:
			fmt.Println("Wrong choice! Try again!")
		}
	}
}

func (e *Explorer) getFullState(stateKey string) string {
	state, ok := e.StateMap[stateKey]
	if !ok {
		return "No such state\n"
	}
	out := fmt.Sprintf("State Key: %s\nState:\n %s\n", stateKey, state.String())
	return out
}

func (e *Explorer) getQValues(state string) string {
	values, ok := e.QTable.GetAll(state)
	if !ok {
		return "No such state in the q table\n"
	}
	if len(values) == 0 {
		return "No values in the q table for the corresponding state\n"
	}
	out := "Q values are:\n"
	for k, v := range values {
		out += fmt.Sprintf("%s: %f\n", k, v)
	}
	return out
}

func (e *Explorer) getInitialStates() string {
	initalStates := make(map[string]int)
	for _, t := range e.Traces {
		if t.Len() == 0 {
			continue
		}
		i := t.States[0]
		if _, ok := initalStates[i.Key]; !ok {
			initalStates[i.Key] = 0
		}
		initalStates[i.Key] += 1
	}
	out := "Initial States are:\n"
	for k, o := range initalStates {
		out += fmt.Sprintf("%s: %d\n", k, o)
	}
	return out
}

func (e *Explorer) header() string {
	return `
Welcome to the q table explorer!
	`
}

func (e *Explorer) prompt() string {
	return `
------------------------------------
Select one of the following options:
1. Show initial state
2. Show QValues
3. Show full state
4. Explore a trace
5. Quit
Enter your choice: `
}

func (e *Explorer) tracePrompt() string {
	return `
---------------------------------------------
Step(s) QValues(d) Prev(p) Last(l) Quit(q): `
}

func (e *Explorer) interactTrace(traceNo int, reader *bufio.Reader) {
	stepCount := 0
	trace := e.Traces[traceNo]
	if trace.Len() == 0 {
		fmt.Println("Empty trace!")
		return
	}
	fmt.Println("---------------------------------------------")
	for {
		s, a, ns, _ := trace.Get(stepCount)
		fmt.Printf("For step %d\nState: %s\nAction:%s\nNextState: %s\n", stepCount+1, s.String(), a.String(), ns.String())
		fmt.Printf("%s", e.tracePrompt())
		optionS, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Invalid input! Try again")
			continue
		}
		fmt.Println("---------------------------------------------")
		option := strings.Replace(optionS, "\n", "", -1)
		switch option {
		case "s":
			if stepCount == trace.Len()-1 {
				fmt.Println("No more steps!")
				continue
			}
			stepCount += 1
		case "d":
			fmt.Printf("%s", e.getQValues(s.Key))
		case "p":
			if stepCount == 0 {
				fmt.Printf("No more steps!")
				continue
			}
			stepCount -= 1
		case "l":
			stepCount = trace.Len() - 1
		case "q":
			return
		default:
			fmt.Println("Invalid option! Try again.")
		}
	}
}

// func (e *Explorer) traceStepPrompt() string {
// 	return `
// Select one of the following options:
// 1. Show full state
// 2. Show q values at state
// 3. Show full action
// 4. Show full next state
// 5. Stop debugging
// Enter your choice: `
// }

// func (e *Explorer) interactTraceStep(traceNo, step int, reader *bufio.Reader) {
// 	trace := e.Traces[traceNo]
// 	s, a, ns, _ := trace.Get(step)
// 	for {
// 		fmt.Printf("%s", e.traceStepPrompt())
// 		optionS, err := reader.ReadString('\n')
// 		if err != nil {
// 			fmt.Println("Invalid input! Try again")
// 			continue
// 		}
// 		option, err := strconv.Atoi(strings.Replace(optionS, "\n", "", -1))
// 		if err != nil {
// 			fmt.Println("Invalid input! Try again")
// 			continue
// 		}
// 		switch option {
// 		case 1:
// 			fmt.Printf("%s", e.getFullState(s.Key))
// 		case 2:
// 			fmt.Printf("%s", e.getQValues(s.Key))
// 		case 3:
// 			fmt.Printf("%s\n", a.String())
// 		case 4:
// 			fmt.Printf("%s", e.getFullState(ns.Key))
// 		case 5:
// 			return
// 		default:
// 			fmt.Println("Invalid option! Try again.")
// 		}
// 	}
// }
