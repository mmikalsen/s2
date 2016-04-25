#!/bin/bash


#ansible tiles -m shell -a 'export DISPLAY=:0 && export GOPATH=$HOME/go && lxterm -bg black -fg white -hold -maximized -e "cd ~/teams/tombak/s2/src && python greencompute.py; bash"' --limit=tile-1-0  

#ansible tiles -m shell -a 'export DISPLAY=:0 && export GOPATH=$HOME/go && lxterm -bg black -fg white -hold -maximized -e "cd ~/teams/tombak/s2/src && go run lb.go; bash"' --limit=tile-0-0 &

#ansible tiles -m shell -a 'export DISPLAY=:0 && export GOPATH=$HOME/go && lxterm -bg black -fg white -hold -maximized -e "cd ~/teams/tombak/s2/src && go run frontend.go; bash"' --limit=tile-0-1 &

#ansible tiles -m shell -a 'export DISPLAY=:0 && export GOPATH=$HOME/go && lxterm -bg black -fg white -hold -maximized -e "cd ~/teams/tombak/s2/src && go run frontend.go; bash"' --limit=tile-0-2 &


#ansible tiles -m shell -a 'export DISPLAY=:0 && export GOPATH=$HOME/go && lxterm -bg black -fg white -hold -maximized -e "cd ~/teams/tombak/s2/src && go run frontend.go; bash"' --limit=tile-0-3 &

ansible tiles -m shell -a 'export DISPLAY=:0 && export GOPATH=$HOME/go && lxterm -bg black -fg white -hold -maximized -e "cd ~/teams/tombak/s2/src && go run client.go; bash"' --limit=tile-2-0 &

