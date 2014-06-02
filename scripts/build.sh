#!/bin/bash
#
# This script builds the application from source.
set -e

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that directory
cd $DIR

# Get the git commit
GIT_COMMIT=$(git rev-parse HEAD)
GIT_DIRTY=$(test -n "`git status --porcelain`" && echo "+CHANGES" || true)

# If we're building on Windows, specify an extension
EXTENSION=""
if [ "$(go env GOOS)" = "windows" ]; then
    EXTENSION=".exe"
fi

GOPATHSINGLE=${GOPATH%%:*}
if [ "$(go env GOOS)" = "windows" ]; then
    GOPATHSINGLE=${GOPATH%%;*}
fi

if [ "$(go env GOOS)" = "freebsd" ]; then
  export CC="clang"
  export CGO_LDFLAGS="$CGO_LDFLAGS -extld clang" # Workaround for https://code.google.com/p/go/issues/detail?id=6845
fi

# Install dependencies
echo "--> Installing dependencies to speed up builds..."
go get \
  -ldflags "${CGO_LDFLAGS}" \
  ./...

# Build CLI!
echo "--> Building CLI..."
cd "${DIR}/cli"
go build \
    -ldflags "${CGO_LDFLAGS} -X main.GitCommit ${GIT_COMMIT}${GIT_DIRTY}" \
    -v \
    -o ../bin/crosby${EXTENSION}
cp ../bin/crosby${EXTENSION} ${GOPATHSINGLE}/bin

# Start Server
echo "--> Building Server..."
cd "${DIR}/server"
go build \
    -ldflags "${CGO_LDFLAGS} -X main.GitCommit ${GIT_COMMIT}${GIT_DIRTY}" \
    -v \
    -o ../bin/server${EXTENSION}
cp ../bin/server${EXTENSION} ${GOPATHSINGLE}/bin

echo "--> Checking that Mongo is Up"
mongo --quiet --eval "'Mongo is Up!'"  # do a simple harmless command of some sort

RESULT=$?   # returns 0 if mongo eval succeeds

if [ $RESULT -ne 0 ]; then
    echo "Starting MongoDB..."
    mongod &
fi

echo "--> Starting Server..."
../bin/server
