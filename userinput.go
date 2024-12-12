package main

import (
	"log"
	"github.com/hajimehoshi/ebiten/v2"
)

type KeyList []ebiten.Key

type ControlState struct {
	down    bool
	press   bool
	release bool
}

type UserInput struct {
	keyDesc  map[string]KeyList
	keyState map[string]*ControlState
	mouseRightState ControlState
	mouseLeftState  ControlState
}

func NewUserInput(keyDesc *map[string]KeyList) *UserInput {
	log.Printf("NewUserInput() %d key descriptors", len(*keyDesc))
	userInput := &UserInput{
		keyDesc: *keyDesc,
		keyState: map[string]*ControlState{},
	}

	for keyName, _ := range *keyDesc {
		userInput.keyState[keyName] = &ControlState{}
	}
	
	return userInput
}

func (userInput *UserInput) handleKeys() {
	for keyName, keys := range userInput.keyDesc {
		userInput.handleKeyPress(keys, userInput.keyState[keyName])
	}
}

func (userInput *UserInput) handleMouse() {
	down := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	userInput.updateControlState(down, &userInput.mouseLeftState)

	down = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	userInput.updateControlState(down, &userInput.mouseRightState)
}

func (userInput *UserInput) updateControlState(isControlDown bool, state *ControlState) {
	state.press = isControlDown && !state.down
	state.release = !isControlDown && state.down
	state.down = isControlDown
}

/*
handleKeyPress centralizes the handling of key presses to reduce redundancy.
*/
func (userInput *UserInput) handleKeyPress(keys KeyList, state *ControlState) {
	down := false
	for _, key := range keys {
		if ebiten.IsKeyPressed(key) {
			down = true
			break
		}
	}

	userInput.updateControlState(down, state)
}

func (userInput *UserInput) isKeyPressed(keyName string) bool {
	state, ok := userInput.keyState[keyName]
	if ok {
		return state.press
	} else {
		return false
	}
}

func (userInput *UserInput) isMouseLeftClick() bool {
	return userInput.mouseLeftState.press
}

func (userInput *UserInput) isMouseRightClick() bool {
	return userInput.mouseRightState.press
}
