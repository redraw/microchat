package lib

type commandID int

const (
	CMD_NICK commandID = iota
	CMD_JOIN
	CMD_ROOMS
	CMD_MSG
	CMD_MSG_USER
	CMD_LIST_MEMBERS
	CMD_WHOIS
	CMD_ME
	CMD_HELP
	CMD_QUIT
)

type command struct {
	id     commandID
	client *client
	args   []string
}
