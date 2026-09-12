#!/bin/sh
#
# SPDX-FileCopyrightText: The Calyx Institute
# SPDX-License-Identifier: Apache-2.0
#

set -eu

image=${IMAGE:-golang:1.26.8}

exec docker run --rm \
	-v "$PWD":/src \
	-w /src \
	--user "$(id -u):$(id -g)" \
	-e GOCACHE=/tmp/gocache \
	-e GOPATH=/tmp/gopath \
	"$image" \
	sh -c 'make'
