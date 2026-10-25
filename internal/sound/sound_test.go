package sound

import "testing"

func TestControllerAdjustsWithinBoundsAndRestoresMuteVolume(t *testing.T) {
	var events []Event
	controller := NewController(PlayerFunc(func(event Event) { events = append(events, event) }))
	if controller.Volume() != DefaultVolume {
		t.Fatalf("default volume = %d, want %d", controller.Volume(), DefaultVolume)
	}
	if got := controller.Adjust(100); got != MaxVolume {
		t.Errorf("increased volume = %d, want %d", got, MaxVolume)
	}
	if got := controller.Adjust(-200); got != 0 {
		t.Errorf("decreased volume = %d, want 0", got)
	}
	controller.Play(Check)
	if len(events) != 0 {
		t.Errorf("muted controller played %v", events)
	}
	if got := controller.ToggleMute(); got != MaxVolume {
		t.Errorf("unmuted volume = %d, want %d", got, MaxVolume)
	}
	controller.Play(Check)
	if len(events) != 1 || events[0] != Check {
		t.Errorf("events = %v, want [check]", events)
	}
}
