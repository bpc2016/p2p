package main

import (
	"testing"
)

// so that we can use app.applications here, invoke db
func setup() (*application, ChatRoom) {

	// Initialize  the application with its dependencies.
	app := &application{
		debug: true,
	}
	cr := ChatRoom{}
	return app, cr
}

// fp.gather returns a list of all related pages from fullpage
func TestHelp(t *testing.T) {
	// initialize ap, cr
	// app, cr := setup()
	_, cr := setup()

	var tests = []struct {
		cmd  string
		rest string
		res  string
	}{
		{"/help", "", ""},
		{"/help", "topic", "topic"},
	}

	// func (cr *ChatRoom) find(cmd string, prs *[]string, rest string) error {

	params := []string{}
	for _, test := range tests {
		cr.find(test.cmd, &params, test.rest)
		if params[0] != test.res {
			t.Errorf("testing find command (%q) = %v, wanted %v", test.cmd, params, test.res)
		}
		// fp, _ := app.baobab.GetFullPage2(ts.addr)
		// // PrintJSON(fp)
		// lst := fp.Gather()
		// sort.Strings(lst)
		// if fmt.Sprintf("%v", lst) != fmt.Sprintf("%v", ts.ads) {
		// 	t.Errorf("gathered addreses of (%q) = %v, wanted %v", ts.addr, lst, ts.ads)
		// }
	}
}
