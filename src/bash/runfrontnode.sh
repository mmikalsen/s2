#/bin/bash -l

#executable
dir="`pwd`"
executable="/frontend.go"

if [ "$1" == "-r" ]; then
    go build $dir$executable
    executable="/frontend"
    
    echo $dir$executable
    echo "Booting Frontends"
    for ((i = 1; i <= 3; i++)); do
        echo "booting compute-10-${i}"
        nohup ssh "compute-10-${i}" bash -c "'$dir$executable'" &
    done
else 
    for ((i = 1; i <= 3; i++)); do
        ssh "compute-10-${i}" bash -c "'pgrep -f 'frontend' | xargs kill -SIGQUIT'"
        if ssh "compute-10-${i}" bash -c ps aux | grep 'frontend' > /dev/null 2>&1 ; then
            echo "failed to kill executable on compute-10-${i}"
        fi
    done
fi
