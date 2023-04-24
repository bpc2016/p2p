package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// so that we can use app.applications here, invoke db
func Setup() (*application, ChatRoom) {

	// Initialize  the application with its dependencies.
	app := &application{
		debug: true,
	}
	cr := ChatRoom{}
	return app, cr
}

func TestHandleC(t *testing.T) {
	// initialize ap, cr
	// app, cr := Setup()
	_, cr := Setup()

	var tests = []struct {
		s        string // command
		cm       string
		to       string
		recevied string
		by       string
		// input -- what gets publ (cmd line) -- where to --
		// any error on recv side? --
	}{
		// {"/fetch everyone\n", "check the json payload\n", "", "check the json payload\n", ""},
		{"/to addr /fetch everyone\n", "/fetch everyone\n", "addr", "check the json payload\n", "sender"},
		// {"/to all /iiam\n", "/iiam\n", "", "/iiam\n", "sender"},
		// {"/who\n", "/to all /iam\n", "", "/iam\n", "sender"},
		// {"/who2\n", "/to me /iam\n", "", "/iam\n", "me"},
	}

	for _, test := range tests {
		// mock console
		ss := test.s // handle modifies this!
		// handle converts test.s --> test.cm
		tto := "" // ditto
		cr.handleCommands(&ss, &tto, nil)
		if ss != test.cm {
			t.Errorf("cm - handlecmnds, with cmd string (%q) = %q, wanted %q", test.s, ss, test.cm)
		}
		if tto != test.to {
			t.Errorf("to - handlecmnds, with cmd string (%q) = %q, wanted %q", test.s, tto, test.to)
		}
		// test inside
		payload := ""
		got, to, err := inside(ss, payload, &cr)
		if err != nil {
			t.Errorf("inside error, w/ cmd string (%q) = %v, wanted %v", test.s, err, nil)
		}
		// got := ""
		if got != test.recevied {
			t.Errorf("received w/ start w/ cmd string (%q) = %q, wanted %q", test.s, got, test.recevied)
		}
		if to != test.by {
			t.Errorf(".. by w/ start w/ cmd string (%q) = %q, wanted %q", test.s, to, test.by)
		}
	}
}

func inside(cmMessage, cmPayload string, cr *ChatRoom) (string, string, error) {
	// is this a remote command?
	if strings.HasPrefix(cmMessage, "/") {

		// fmt.Printf("\tseeing msg %q with payload %q\n", cmMessage, cmPayload)

		if !cr.validCommand(cmMessage) {
			// fmt.Printf("\tinvalid commands : %q\n", cmMessage)
			return "", "", fmt.Errorf("invalid commands : %q", cmMessage)
		}
		cmTo := ""
		// prep, if there is a payload, it is in cm.Payload
		if _, err := cr.handleCommands(&cmMessage, &cmTo, nil); err != nil {
			//fmt.Printf("\thandlecomands error: %v\n", err)
			return "", "", fmt.Errorf("handlecomands error: %v", err)
			//return err
		}

		// fmt.Printf("== caught %q going to %q w/ payload %v\n", cm.Message, cm.Sender(), cm.Payload) // ****

		/*
			// otherwise , just publish it, back to sender
			fmt.Pif err := cr.Publish(cmMessage, cm.Sender(), cmPayload); err != nil {
				fmt.Printf("publish error: %v/n", err)
			}
		*/
		return cmMessage, "sender", nil
	}
	// fmt.Printf("msg %q has no leading '/'\n", cmMessage)
	return cmMessage, "", nil
}

func TestSample(t *testing.T) {
	// initialize ap, cr
	// app, cr := Setup()
	_, cr := Setup()

	var tests = []struct {
		s      string // command
		to     string
		sample string
		res    string
		cm     string
	}{
		{"/fetch everyone\n", "aguy34", "everyone", `{"Tag":"everyone","Data":"EVERYONE"}`, "check the json payload\n"},
		// {"/help", "topic", "topic"},
	}
	for _, test := range tests {
		//cr.find(test.cmd, &params, test.rest)
		if got := string(sampleFetch(test.sample)); got != test.res {
			t.Errorf("sampleFetch (%q) = %v, wanted %q", test.sample, got, test.res)
		}
		// test handlecommands
		ss := test.s   // handle modifies this!
		tto := test.to // ditto
		p, err := cr.handleCommands(&ss, &tto, nil)
		if err != nil {
			t.Errorf("handlecmnds, with cmd string (%q) = %v, wanted %v", test.s, err, nil)
		}
		resstr := string(p) //fmt.Sprintf("%v", p)
		if resstr != test.res {
			t.Errorf("handlecmnds payload, w/ cmd string (%q) = %q, wanted %v", test.s, resstr, test.res)
		}
		// mock console
		ss = test.s // handle modifies this!
		if err := console(&ss, &cr); err != nil {
			t.Errorf("console error, w/ cmd string (%q) = %v, wanted %v", test.s, err, nil)
		}
		// handle converts test.s --> test.cm
		if ss != test.cm {
			t.Errorf("handle.. converts cmd string (%q) to %s, wanted %q", test.s, ss, test.cm)
		}
		// mock the transmitted info, first have the message structure,
		// look at cr.Publish
		m := ChatMessage{
			Message:    test.cm,
			To:         tto,
			Payload:    p,
			SenderID:   "",
			SenderNick: "",
		}
		msgBytes, err := json.Marshal(m)
		if err != nil {
			t.Errorf("json marshal error %q", err)
		}
		// which is consumed by readLoop
		cm := new(ChatMessage)
		err = json.Unmarshal(msgBytes, cm)
		if err != nil {
			t.Errorf("json Unmarshal error %q", err)
		}
		// check:
		PrintJSON(cm.Payload)       // rubbish: eyJUYWciOiJldmVyeW9uZSIsIkRhdGEiOiJFVkVSWU9ORSJ9
		println(string(cm.Payload)) // this is ok: {"Tag":"everyone","Data":"EVERYONE"}
		var vi interface{}
		err = json.Unmarshal(cm.Payload, &vi)
		if err != nil {
			t.Errorf("json Unmarshal error %q", err)
		}
		PrintJSON(vi)
	}
}

func console(s *string, cr *ChatRoom) error {
	reloc := regexp.MustCompile("^/") //(`^\/`)

	//in case we have private messages
	to := ""            // default: public
	payload := []byte{} // empty

	fmt.Printf("command string s: %q\n", *s)

	if reloc.MatchString(*s) {
		p, err := cr.handleCommands(s, &to, nil)
		if err != nil {
			if err != errSkip {
				fmt.Printf("%v\n", err)
			}
			return err
		}
		payload = p
	}

	fmt.Printf("--- publishing %q to %q with payload %q\n", *s, to, string(payload))

	// publish
	// if err := cr.Publish(s, to, payload); err != nil {
	// 	return err
	// }
	return nil
}

/*
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
*/
