#!/bin/sh -l

if [ -z "${GITHUB_PAT}" ]
then
    echo "The GITHUB_PAT environment variable is not defined."
    exit 1
fi

RELEASE_BRANCH="$1"

/auto-semver-tag exec "${GITHUB_REPOSITORY}" "${RELEASE_BRANCH}" "${GITHUB_SHA}" "${GITHUB_EVENT_PATH}"
