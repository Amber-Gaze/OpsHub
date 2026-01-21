#!/usr/bin/env bash 

set -e

ROOT=$(cd $(dirname $0)/.. && pwd)
cd $ROOT

SYSTEM=$(uname -s)
if [[ "$SYSTEM" != "Linux" ]]; then
    echo "build only support linux"
    exit 1
fi

PROJECT=`basename $ROOT`
PROGRAM=$PROJECT
if [[ "$VERSION" == "" ]]; then
    VERSION=$(cat CHANGELOG.md | grep -E "## \[([0-9]+)\.([0-9]+)\.([0-9]+)\]" | head -1 | awk '{print $2}' | tr -d '[]')
fi
GITHASH=$(git rev-parse --short=8 HEAD)
DATESTR=$(date '+%Y%m%d%H%M%S')
 

function build() { 
    rm -rf output 
    mkdir -p output/{data,conf,bin,logs}
    cp build/bin/*  output/bin/
    cp build/conf/* output/conf/

    CGO_ENABLED=0 

    echo "building $PROGRAM version:$VERSION git:$GITHASH date:$DATESTR"
     
    FLAGS="-X 'main.version=$PROGRAM $VERSION ($GITHASH $DATESTR)'"
    #-X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn
    go build -trimpath -ldflags "$FLAGS" \
        -o $ROOT/output/bin/$PROGRAM \
        ./cmd/api/main.go


    if [ $? -ne 0 ]; then
        echo "go build failed: $?"
        exit 1
    fi

    # calculate md5
    md5sum output/bin/$PROGRAM > output/md5sum.file
}


function pack() {
    PACKAGE="$PROGRAM-$VERSION-$GITHASH-$DATESTR"
    cp -rf output $PACKAGE
    tar czvf $PACKAGE.tar.gz $PACKAGE
    rm -rf $PACKAGE
}

function main() {
    build

    if [[ "$1" == "pack" ]]; then
        pack
    fi
}

main $1