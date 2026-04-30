package notifications

import (
	"fmt"
	// "os"
	"os/exec"
	// "slices"
	// "strconv"
)

const (
	AppTitle = "Assets Tracker"
)

type Notification struct {
	TimeToExpire int
	StockTitle string
	Body string
}

type Notifier interface {
	Notify(Notification) error
}

type LibNotifyNotifier struct {}

func NewLibNotifyNotifier() *LibNotifyNotifier {
	return &LibNotifyNotifier{}
}

func (l *LibNotifyNotifier) Notify(n Notification) error {

	time_to_expire := fmt.Sprintf("--expire-time=%d", n.TimeToExpire)
	cmd := exec.Command("/usr/bin/notify-send", time_to_expire, n.StockTitle, n.Body)

	err := cmd.Start()

	if err != nil {
		fmt.Printf("ERROR: %v.\n", err)
		return fmt.Errorf("ERROR: %v.\n", err)
	}

	err = cmd.Wait()

	if err != nil {
		fmt.Printf("ERROR: %v.\n", err)
		return fmt.Errorf("ERROR: %v.\n", err)
	}

	return nil
}

type HyprlandNotifier struct{}

func NewHyprlandNotifier() *HyprlandNotifier {
	return &HyprlandNotifier{}
}

func (hypr *HyprlandNotifier) Notify(n Notification) error {
	time_to_expire := fmt.Sprintf("%d", n.TimeToExpire)
	notification_icon := "1" // 1 = Info icon
	text_color := "rgb(ff1ea3)"
	treated_body := fmt.Sprintf("fontsize:22  %s", n.Body)
	cmd := exec.Command("/usr/bin/hyprctl", "notify", notification_icon, time_to_expire, text_color, treated_body)

	err := cmd.Start()

	if err != nil {
		fmt.Printf("ERROR: %v.\n", err)
		return fmt.Errorf("ERROR: %v.\n", err)
	}

	err = cmd.Wait()

	if err != nil {
		fmt.Printf("ERROR: %v.\n", err)
		return fmt.Errorf("ERROR: %v.\n", err)
	}

	return nil
}
