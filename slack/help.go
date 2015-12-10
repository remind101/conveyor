package slack

const helpText = `Enable conveyor on a repo: /conveyor enable REPO
Build a branch on a repo: /conveyor build REPO@BRANCH`

var Help = replyHandler(helpText)
