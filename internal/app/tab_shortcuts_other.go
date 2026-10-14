//go:build !darwin

package app

func (app *Application) installWindowTabShortcuts() func() { return func() {} }
