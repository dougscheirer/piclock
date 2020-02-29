#!/bin/bash

die()
{
  echo "$*"
  exit 1
}

# do a git pull, a go test/build, and start the service
git pull || die "failed to perform git pull"
go test || die "go tests failed!"
go build || die "go build failed!"

systemctl restart piclock.service || die "failed to restart service"