package panictest

import "fmt"

func Recovered(f func()) (refused any) {
	defer func() { refused = recover() }()
	f()
	return nil
}

func Message(f func()) (msg string, refused bool) {
	recovered := Recovered(f)
	if recovered == nil {
		return "", false
	}
	return fmt.Sprint(recovered), true
}
