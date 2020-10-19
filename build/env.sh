#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
projectdir="$workspace/src/github.com/hbtc-chain"
if [ ! -L "$projectdir/chainnode" ]; then
    mkdir -p "$projectdir"
    cd "$projectdir"
    ln -s ../../../../../. chainnode
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$projectdir/chainnode"
PWD="$projectdir/chainnode"

# Launch the arguments with the configured environment.
exec "$@"
