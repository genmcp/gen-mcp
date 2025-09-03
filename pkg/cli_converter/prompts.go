package cli_converter

var IsSubCommandPrompt = `Given below is the command and man page for CLI based utility.
You must return true if the man page lists any sub-commands, otherwise return false.
Do NOT consider example, usages, description, arguments, flags, options as sub-commands.

Example 1:
# Example 1:
# Man page for "git"
#
# GIT(1)                        Git Manual                       GIT(1)
#
# NAME
#        git - the stupid content tracker
#
# SYNOPSIS
#        git [--version] [--help] [-C <path>] [-c <name>=<value>] [--exec-path[=<path>]]
#            [--html-path] [--man-path] [--info-path] [-p | --paginate | -P | --no-pager]
#            [--no-replace-objects] [--bare] [--git-dir=<path>] [--work-tree=<path>]
#            [--namespace=<name>] <command> [<args>]
#
# DESCRIPTION
#        Git is a fast, scalable, distributed revision control system.
#
# COMMANDS
#        The most commonly used git commands are:
#            add        Add file contents to the index
#            commit     Record changes to the repository
#            push       Update remote refs along with associated objects
#            pull       Fetch from and integrate with another repository or a local branch
#            status     Show the working tree status
#
#        See 'git help <command>' to read about a specific subcommand.
#
# OPTIONS
#        --version
#            Prints the git suite version that the git program came from.
#
#        --help
#            Prints the synopsis and a list of commands.
Output: bool_value=true

Example 2:
# Example 2:
# Man page for "ls"
#
# LS(1)                        User Commands                       LS(1)
#
# NAME
#        ls - list directory contents
#
# SYNOPSIS
#        ls [OPTION]... [FILE]...
#
# DESCRIPTION
#        List  information  about the FILEs (the current directory by default).
#
# OPTIONS
#        -a, --all
#            do not ignore entries starting with .
#        -l
#            use a long listing format
#        -h, --human-readable
#            with -l, print sizes in human readable format (e.g., 1K 234M 2G)
#
# EXAMPLES
#        ls -l
#            List files in the long format.
#
#        ls -a
#            List all files including hidden files.
#
# There are no sub-commands in this man page.
#
Output: bool_value=false
`
