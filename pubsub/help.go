package main

// simply print our item
func gethelp(it string) {
	println(help[it])
}

var help = map[string]string{
	"0": `
-----------------------------
Welcome to our chatroom!
Best to start with a nickname: if you haven't already, 
Restart and use a nickname, like so:
	/quit
	./chat -nick <yournickname>

type /help for a list of topics.
-----------------------------`,
	"": "1. what this app does\n2. flags",
}
