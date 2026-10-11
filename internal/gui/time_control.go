package gui

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gdamore/tcell/v2"
)

type timeControl struct {
	initial   time.Duration
	increment time.Duration
}

func (c timeControl) isTimed() bool {
	return c.initial > 0
}

type timeControlChoice struct {
	label   string
	control timeControl
	custom  bool
}

var timeControlChoices = []timeControlChoice{
	{label: "1+0", control: timeControl{initial: time.Minute}},
	{label: "2+1", control: timeControl{initial: 2 * time.Minute, increment: time.Second}},
	{label: "3+0", control: timeControl{initial: 3 * time.Minute}},
	{label: "3+2", control: timeControl{initial: 3 * time.Minute, increment: 2 * time.Second}},
	{label: "5+0", control: timeControl{initial: 5 * time.Minute}},
	{label: "5+3", control: timeControl{initial: 5 * time.Minute, increment: 3 * time.Second}},
	{label: "10+0", control: timeControl{initial: 10 * time.Minute}},
	{label: "10+5", control: timeControl{initial: 10 * time.Minute, increment: 5 * time.Second}},
	{label: "15+10", control: timeControl{initial: 15 * time.Minute, increment: 10 * time.Second}},
	{label: "30+0", control: timeControl{initial: 30 * time.Minute}},
	{label: "30+20", control: timeControl{initial: 30 * time.Minute, increment: 20 * time.Second}},
	{label: "Unlimited time"},
	{label: "Custom", custom: true},
}

type customTimeField uint8

const (
	customMinutes customTimeField = iota
	customIncrement
)

func (s *menuState) moveTimeControlSelection(delta int) {
	s.timeControlSelected = (s.timeControlSelected + delta + len(timeControlChoices)) % len(timeControlChoices)
}

func (s *menuState) activateTimeControl() bool {
	choice := timeControlChoices[s.timeControlSelected]
	if choice.custom {
		s.page = customTimeControlMenu
		s.customFocus = customMinutes
		s.customMessage = ""
		return false
	}
	s.timeControl = choice.control
	s.result = LocalGame
	return true
}

func (s *menuState) handleCustomTimeControlKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyEsc:
		s.page = timeControlMenu
		s.customMessage = ""
	case tcell.KeyTab:
		s.customFocus = (s.customFocus + 1) % 2
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		field := s.customField()
		if len(*field) > 0 {
			*field = (*field)[:len(*field)-1]
		}
	case tcell.KeyEnter:
		return s.startCustomTimeControl()
	case tcell.KeyRune:
		if event.Rune() >= '0' && event.Rune() <= '9' {
			field := s.customField()
			*field += string(event.Rune())
		}
	}
	return false
}

func (s *menuState) customField() *string {
	if s.customFocus == customIncrement {
		return &s.customIncrement
	}
	return &s.customMinutes
}

func (s *menuState) startCustomTimeControl() bool {
	minutes, minuteError := strconv.Atoi(s.customMinutes)
	increment, incrementError := strconv.Atoi(s.customIncrement)
	if minuteError != nil || minutes < 1 || minutes > 90 {
		s.customMessage = "Minutes must be between 1 and 90"
		return false
	}
	if incrementError != nil || increment < 0 || increment > 60 {
		s.customMessage = "Increment must be between 0 and 60"
		return false
	}
	s.timeControl = timeControl{
		initial:   time.Duration(minutes) * time.Minute,
		increment: time.Duration(increment) * time.Second,
	}
	s.result = LocalGame
	return true
}

func drawTimeControlMenu(screen tcell.Screen, state menuState) {
	width, height := screen.Size()
	x := (width - 28) / 2
	top := (height - 22) / 2
	y := top + 3
	drawCentered(screen, top, "Play > Local game", titleStyle)
	drawCentered(screen, top+1, "Please select a time control", mutedStyle)

	choiceIndex := 0
	drawTimeControlSection(screen, x, y, "🚅 Bullet", state, &choiceIndex, 2)
	y += 3
	drawTimeControlSection(screen, x, y, "⚡ Blitz", state, &choiceIndex, 4)
	y += 5
	drawTimeControlSection(screen, x, y, "⏱ Rapid", state, &choiceIndex, 3)
	y += 4
	drawTimeControlSection(screen, x, y, "🕰 Classical", state, &choiceIndex, 3)
	y += 4
	drawTimeControlChoice(screen, x, y, "🆓 "+timeControlChoices[choiceIndex].label, state.timeControlSelected == choiceIndex)
	drawCentered(screen, top+21, "Up/Down to select  Enter to choose", mutedStyle)
}

func drawTimeControlSection(screen tcell.Screen, x, y int, title string, state menuState, choiceIndex *int, choices int) {
	drawString(screen, x, y, title, titleStyle)
	for row := 0; row < choices; row++ {
		index := *choiceIndex
		drawTimeControlChoice(screen, x, y+row+1, "  "+timeControlChoices[index].label, state.timeControlSelected == index)
		*choiceIndex = *choiceIndex + 1
	}
}

func drawTimeControlChoice(screen tcell.Screen, x, y int, label string, selected bool) {
	style := backgroundStyle
	if selected {
		style = selectedStyle
	}
	drawPaddedString(screen, x, y, 28, label, style)
}

func timeControlChoiceAt(x, y, width, height int) (int, bool) {
	menuX := (width - 28) / 2
	if x < menuX || x >= menuX+28 {
		return 0, false
	}
	top := (height - 22) / 2
	rows := []int{top + 4, top + 5, top + 7, top + 8, top + 9, top + 10, top + 12, top + 13, top + 14, top + 16, top + 17, top + 18, top + 19}
	for index, row := range rows {
		if y == row {
			return index, true
		}
	}
	return 0, false
}

func drawCustomTimeControl(screen tcell.Screen, state menuState) {
	width, height := screen.Size()
	x := (width - 40) / 2
	y := height/2 - 4
	drawString(screen, x, y, "Custom time control", titleStyle)
	drawCustomTimeField(screen, x, y+2, "Minutes per side", state.customMinutes, state.customFocus == customMinutes)
	drawCustomTimeField(screen, x, y+4, "Increment seconds", state.customIncrement, state.customFocus == customIncrement)
	drawString(screen, x, y+6, "[Back]  Enter: start", mutedStyle)
	if state.customMessage != "" {
		drawString(screen, x, y+7, state.customMessage, mutedStyle)
	}
}

func drawCustomTimeField(screen tcell.Screen, x, y int, label, value string, focused bool) {
	style := backgroundStyle
	if focused {
		style = selectedStyle
	}
	drawString(screen, x, y, fmt.Sprintf("%s: [%5s]", label, value), style)
}

func customTimeFieldAt(x, y, width, height int) (customTimeField, bool) {
	fieldX := (width - 40) / 2
	fieldY := height/2 - 4
	if x < fieldX || x >= fieldX+40 {
		return 0, false
	}
	switch y {
	case fieldY + 2:
		return customMinutes, true
	case fieldY + 4:
		return customIncrement, true
	}
	return 0, false
}

func customBackAt(x, y, width, height int) bool {
	fieldX := (width - 40) / 2
	return y == height/2+2 && x >= fieldX && x < fieldX+6
}
