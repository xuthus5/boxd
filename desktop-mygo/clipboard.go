package main

import "github.com/egoist/mygo"

func mygoClipboardWrite(text string) error {
	mygo.Clipboard.WriteText(text)
	return nil
}
