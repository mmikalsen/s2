#/bin/bash -l

#executable
dir="`pwd`"
executable="client.go"

if [ "$1" == "-r" ]; then
    go build $executable
    executable="client"

    echo "starting on $2 nodes"

    for ((i = 2; i <= $2; i++)); do
        for ((j = 1; j <= 3; j++)); do
            echo "starting compute-${i}-${j}"
            nohup ssh "compute-${i}-${j}" bash -c "'cd $dir; ./$executable'" &
        done
    done

else 
    for ((i = 2; i <= $1; i++)); do
        for ((j = 1; j <= 3; j++)); do
            ssh "compute-${i}-${j}" bash -c "'pgrep -f 'client' | xargs kill -SIGQUIT'"
            if ssh "compute-10-${i}" bash -c ps aux | grep 'client' > /dev/null 2>&1 ; then
              echo "failed to kill executable on compute-${i}-${j}"
            fi
        done
    done
fi
