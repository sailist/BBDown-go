#!/bin/bash
commit_msg_file=$1
commit_msg=$(head -n1 "$commit_msg_file")
pattern="^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(.+\))?!?: .+"
if ! echo "$commit_msg" | grep -qE "$pattern"; then
    echo "ERROR: Commit message does not follow Conventional Commits."
    echo "Expected format: <type>[(scope)]: <description>"
    exit 1
fi
