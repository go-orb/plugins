#!/usr/bin/env bash

RUN_PWD="${PWD}"

pushd "${1}"; 
for package in $(grep --null -E "^replace\s+github.com/go-orb/[\/a-zA-Z\-]+ => .*$" go.mod); do
    if [[ "${package:0:18}" != "github.com/go-orb/" ]]; then
        continue;
    fi;

    echo go mod edit -dropreplace="${package}";
    go mod edit -dropreplace="${package}";
    # We should replace @main with @latest once we have releases.
    go get -u "${package}@main"
done;
popd >/dev/null;