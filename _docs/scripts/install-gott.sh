#!/usr/bin/env bash 

REPO_URL='https://github.com/oimyounis/gott.git'
README_URL='https://raw.githubusercontent.com/oimyounis/gott/master/README.md'
SRC_PATH=$GOPATH/src
CLONE_PATH=$GOPATH/src/gott
MAIN_PATH=$CLONE_PATH/main

install_deps()
{
    echo "Installing dependencies..."

    for (( i=1; i<=$#; i++))
    {
        eval val='$'$i
        echo package $i: $val;
        if ! go get $val; then
            echo "An error occurred while installing dependency: $val"
            exit
        fi
    }
}

start_gott()
{
    if [ -d $MAIN_PATH ]; then
        echo ""
        read -s -n 1 -p "Do you wish to run GOTT now? [press Y to agree or any key to exit] " yn
        echo ""

        case $yn in
            [Yy]*)
                cd $MAIN_PATH
                go run main.go
            ;;
            *)
                exit
            ;;
        esac
    fi
}

clone_repo()
{
    echo "Cloning GOTT repo in $SRC_PATH"
    cd $SRC_PATH
    if git clone $REPO_URL; then
        start_gott
    fi
}

fetch_deps()
{
    echo "Fetching dependency list from repo.."
    DEPS=$(curl -s $README_URL | grep 'go get' | sed 's/\$ go get //g')
    echo $DEPS | sed -e $'s/ /\\\n/g'

    read -s -n 1 -p "Do you wish to install dependencies? [press Y to agree or any key to skip] " yn
    echo ""

    case $yn in
        [Yy]*)
            install_deps $DEPS
            clone_repo
        ;;
        *)
            clone_repo
        ;;
    esac
}

if [[ -z "${GOPATH}" ]]; then
    echo "GOPATH is not set. Please set it first before continuing. Aborting.."
    echo "You can check https://golang.org/doc/gopath_code#GOPATH for more info."
elif ! hash go 2>/dev/null; then
    echo "Could not find Go in PATH. Make sure it is installed and reachable. Aborting.."
elif ! hash git 2>/dev/null; then
    echo "Could not find Git in PATH. Make sure it is installed and reachable. Aborting.."
elif [ -d $CLONE_PATH ]; then
    echo "GOTT already exists at $CLONE_PATH."
    echo "Please delete/move it first before installing the latest version. Aborting.."
else
    fetch_deps
fi
