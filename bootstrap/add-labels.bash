#!/usr/bin/env bash

set -e

if [ -z "$1" ] || [ -z "$2" ]; then
	echo "Usage: $0 <github_account_name> <repo_name>"
	exit 1
fi

ACCOUNT_NAME="$1"
REPO_NAME="$2"

i=0
while IFS= read -r line; do
	((i++))
	if [ "$i" -eq 1 ]; then
		continue
	fi
	IFS=',' read -ra values <<<"$line"
	name="${values[0]}"
	description="${values[1]}"
	color="${values[2]}"
	echo "Creating '$name' label..."
	gh label --repo "$ACCOUNT_NAME/$REPO_NAME" create "$name" --description "$description" --color "$color"
done <./bootstrap/labels.csv
