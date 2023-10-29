FOR ROB TO DO:

- if duplicate parsed file names, present the user in CLI with a list of options to choose from.

- if file paths are referenced, pre-prompt with something like "anything that is updated via code, respond in the format of 'file: <path>'\n**{code}**"

- on output from ai, make sure that the resposne contains "file: <path>'\n--TGPT{code}--ETGPT" otherwise the user hit max response tokens or total tokens and you need to prompt them to edit their config with an error message.

- show in the easiest/friendlist way "what do you want to do with the code output?" potential options are (every prompt should have a "cancel" option or on complete, goes back to the main prompt):
  1. copy to clipboard (use a go package)
  2. show the difference between the outputted code for a file and the current code in the file (use a go package or git diff to do this)
  3. save to file (with path entry prompt equal to the file path in the response with a confirmation prompt if file already exists)
  4. exit (using the -c) to return to main AI prompting



  