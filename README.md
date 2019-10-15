# simple-hypervisor

just dead simple processes hypervisor

## how to install

    go install -v github.com/nordicdyno/simple-hypervisor/cmd/...

## how to use

start server:

    sh-srv

show services list:

    sh-ctl list

add service:

    sh-ctl add

## todo

* set ports via command line
* http API docs
* logs redirection
* more sh-ctl commands
* pretty output
* `wait service start/stop` command
