#!/usr/bin/env bash

RUN_PWD="${PWD}"

pushd "${1}"; 
for package in $(grep --null -E "^(require)?\s+github.com/go-orb/[\/a-zA-Z\-]+" go.mod); do
    if [[ "${package:0:18}" != "github.com/go-orb/" ]]; then
        continue;
    fi;

    noprefix=${package:18};

    mycwd="${RUN_PWD}/${1}";
    if [[ "${1:0:2}" == "./" ]]; then
        mycwd="${RUN_PWD}/${1:2}"
    fi

    target="";
    if [[ "${noprefix:0:8}" == "plugins/" ]]; then
        target="${RUN_PWD}/${noprefix:8}";
    else
        target="${RUN_PWD}/../${noprefix}";
    fi

    realpath=$(realpath --relative-to="${mycwd}" "${target}");

    echo go mod edit -replace="${package}=${realpath}";
    go mod edit -replace="${package}=${realpath}";
done;
popd >/dev/null;