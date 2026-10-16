package notify

import toast "git.sr.ht/~jackmordaunt/go-toast/v2"

// AppID is the name shown in the Windows Action Center.
const AppID = "autosave-games"

// ToastSender shows Windows toast notifications.
type ToastSender struct{}

// Send implements Sender.
func (ToastSender) Send(title, body string) error {
	n := toast.Notification{AppID: AppID, Title: title, Body: body}
	return n.Push()
}
