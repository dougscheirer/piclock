#!/bin/bash

die()
{
  echo "$*"
  exit 1
}


GO=/home/pi/gvm/go

echo "Version is $($GO version)"

# do a git pull, a go test/build, and start the service
su pi -c "git pull" || die "failed to perform git pull"
su pi -c "$GO test -v" >> ./test-output.txt 2>&1

if [ ! $? -eq 0 ] ; then 
  die "go tests failed!"
fi

su pi -c "$GO build" || die "go build failed!"

systemctl restart piclock.service || die "failed to restart service"
