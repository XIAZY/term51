//go:build !qnx

package app

// Run serves the text front end until the last shell exits (the pixel front
// end needs the phone's window). It returns the reason it ended.
func (a *App) Run() error {
	return a.runText()
}
