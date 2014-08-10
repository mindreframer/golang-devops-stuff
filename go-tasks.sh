#!/bin/bash


# managing project only goenvs
goenv_on(){
  if [ $# -eq 0 ]; then
    _GOPATH_VALUE="${PWD}/.goenv"
  else
    cd $1 ; _GOPATH_VALUE="${PWD}/.goenv" ; cd -
  fi
  if [ ! -d $_GOPATH_VALUE ]; then
    mkdir -p "${_GOPATH_VALUE}/site"
  fi
  export _OLD_GOPATH=$GOPATH
  export _OLD_PATH=$PATH
  export GOPATH=$_GOPATH_VALUE/site
  export PATH=$PATH:$GOPATH/bin
}
alias goenv_off="export GOPATH=$_OLD_GOPATH ; export PATH=$_OLD_PATH ; unset _OLD_PATH ; unset _OLD_GOPATH"


# managing go deps
go_get_pkg(){
  if [ $# -eq 0 ]; then
    if [ -f "$PWD/go-get-pkg.txt" ]; then
      PKG_LISTS="$PWD/go-get-pkg.txt"
    else
      touch "$PWD/go-get-pkg.txt"
      echo "Created GoLang Package empty list $PWD/go-get-pkg.txt"
      echo "Start adding package paths as separate lines." && return 0
    fi
  else
    PKG_LISTS=($@)
  fi
  for pkg_list in $PKG_LISTS; do
    cat $pkg_list | while read pkg_path; do
      echo "fetching golag package: go get ${pkg_path}";
      echo $pkg_path | xargs go get
    done
  done
}


_OLD_PWD=$PWD
cd $(dirname $0)
  goenv_on

if [[ $# -ne 1 ]]; then
  echo "Use it wisely..."
  echo "Install tall Go lib dependencies: '$0 deps'"
  echo "Run all Tests: '$0 test'"
  exit 1

elif [[ "$1" == "deps" ]]; then
  go_get_pkg

elif [[ "$1" == "test" ]]; then
  $0 bin
  echo
  echo "~~~~~Test Pieces~~~~~"
  go test ./...
  echo
  echo "~~~~~Test Features~~~~~"
  for feature_test in `ls ./tests/go*_client.go`; do
    echo ">> Testing: "$feature_test
    ./bin/goshare_daemon -daemon=start -dbpath=/tmp/GOSHARE.TEST.DB
    go run $feature_test
    ./bin/goshare_daemon -daemon=stop
    rm -rf /tmp/GOSHARE.TEST.DB
  done

elif [[ "$1" == "wiki" ]]; then
  $0 bin
  echo
  echo "~~~~~Visit wiki at GoShare HTTP~~~~~"
  echo "~~~~~   http://0.0.0.0:9999    ~~~~~"
  ./bin/goshare_server

elif [[ "$1" == "bin" ]]; then
  bash $0 deps
  mkdir -p ./bin
  cd ./bin
  for go_code_to_build in `ls ../zxtra/goshare_*.go`; do
    echo "Building: "$go_code_to_build
    go build $go_code_to_build
  done

fi

cd $_OLD_PWD
