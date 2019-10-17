# process chief

Process Chief (PC) - is a dead simple processes hypervisor. Contains utils and Go library for processes control.

## how to install

    go install -v github.com/nordicdyno/process-chief/cmd/...

## components

* pc-srv - server
* pc-ctl - CLI control util (client)
* pc-log - handy logger with ability to reopen logs by signal, useful for log rotation (optional)

## how to use utils

start server:

    pc-srv

show services list:

    pc-ctl list

add service:

    pc-ctl add -h

## how to use library

## todo

* add design rationales to README
* docs
* tests
* set server listen ports via command line
* http API docs / examples
* pretty output
* `wait process` command 
