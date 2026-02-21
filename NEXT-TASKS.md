# Next tasks for yolo-agents

- installer!!! we must have one, so we will be able to install it globally and use. First it should be make install, then it should become an installation script, that will automatically download and install prebuilt version for user's system
- provied task id we should be able to calculate the most efficient concurrency level. There should be "auto" default setting for it
- right now running the agent is kind of hard because of all the piping and everyhing. I thinks that we should have `mode` flag in the runner so if it set to ui it will fork to the ui process and redirect oupit there. This should be configurable through the config file.
- right now in panel completed tasks are not marked in any way, and they should be
- rignt now completion counter in status bar is broken, and it should defenitely work
- agent thoughts and messages are not shown
- it would be nice to have logs by task groupped and browseable in TUI
- we should analyze the task to see if it is full, enough, clear etc. and if not - put a comment and do not do it until we fully understand how is it working
- we should use more clever agent to do PR review
- we should write tests first, use agent to see if they are enough etc., and only after that we should start implementation
- redesingn of TUI should be done, it is really ugly now
- we should be able to specify config for each task with sane defaults, BUT we should really be able to change model, skillset, tools etc. for the task because there could be non coding task also
- in general it would be nice to use some kind of smarter model to recover from any type errors or failures
- agent should log it's actions and decisions to the ones who come after us will know why tings are the way they are



