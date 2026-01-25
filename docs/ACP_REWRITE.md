We should utilize opencode's support of [ACP protocol](https://agentcommunicationprotocol.dev/core-concepts/agent-run-lifecycle) to run it instead of relying on jsonl logs, console output and other tricks.

Please use go [SDK](https://github.com/ironpark/acp-go) to implement go ACP client for OpenCode that will run our tasks instead of trying to do it through CLI.

As a result we should have yolo-runner behave like this
 - pick up next task using beads
 - start opencode in acp mode (please consult `opencode --help` and https://opencode.ai on how to do so)
 - connect to it through ACP protocol
 - select YOLO agent and issue the prompt to execute the task
 - monitor progress through ACP messages
 - if opencode asks for premissions alway grant them and log requests
 - if opencode asks for anythng else tell it to decide itself and log the request
 - after the task is finished log it as complete and proceed to the next task
 
 Please analyze this task, create bead epic and populate it with tasks small enough to be properly tested, link them using beads so the will become a dependency graph. Please make writing test separate tasks. Please follow strict TDD pattern when implementing this. 
